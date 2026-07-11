package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/nimbuscore/apps/workspace-manager/internal/snapshot"
	"github.com/nimbuscore/pkg/api"
	"github.com/nimbuscore/pkg/operator"
	"github.com/nimbuscore/pkg/workspace"
)

type Manager struct {
	cfg            Config
	k8s            kubernetes.Interface
	dynamic        dynamic.Interface
	consumer       *workspace.Consumer
	publisher      *workspace.Publisher
	snapshots      *snapshot.Service
}

type Config struct {
	Ingress         workspace.IngressConfig
	IngressHost     string
	SnapshotEnabled bool
	IdleTimeout     int
}

func New(
	cfg Config,
	k8s kubernetes.Interface,
	dynamic dynamic.Interface,
	rmqCh *amqp.Channel,
	snapshots *snapshot.Service,
	publisher *workspace.Publisher,
) *Manager {
	consumer := workspace.NewConsumer(rmqCh, "nimbuscore.workspace", "workspace-manager")

	mgr := &Manager{
		cfg:       cfg,
		k8s:       k8s,
		dynamic:   dynamic,
		consumer:  consumer,
		publisher: publisher,
		snapshots: snapshots,
	}

	consumer.Handle(workspace.EventWorkspaceCreate, mgr.handleCreate)
	consumer.Handle(workspace.EventWorkspaceStart, mgr.handleStart)
	consumer.Handle(workspace.EventWorkspaceStop, mgr.handleStop)
	consumer.Handle(workspace.EventWorkspaceDelete, mgr.handleDelete)
	consumer.Handle(workspace.EventWorkspaceSnapshot, mgr.handleSnapshot)
	consumer.Handle(workspace.EventWorkspaceTimeout, mgr.handleTimeout)
	consumer.Handle(workspace.EventWorkspacePrebuild, mgr.handlePrebuild)

	return mgr
}

func (m *Manager) Start(ctx context.Context) error {
	slog.Info("workspace manager listening for events")
	return m.consumer.Start(ctx)
}

func (m *Manager) handleCreate(ctx context.Context, event workspace.Event) error {
	slog.Info("workspace create event", "workspace_id", event.WorkspaceID)

	var wsData api.Workspace
	if err := json.Unmarshal(event.Payload, &wsData); err != nil {
		slog.Error("payload unmarshal failed", "error", err, "payload", string(event.Payload))
		return fmt.Errorf("failed to unmarshal workspace payload: %w", err)
	}

	p := make([]operator.PortRule, len(wsData.Ports))
	for i, port := range wsData.Ports {
		p[i] = operator.PortRule{
			Port:      port.Port,
			Protocol:  port.Protocol,
			Subdomain: port.Subdomain,
		}
	}

	wsOperator := &operator.Workspace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operator.GroupName + "/" + operator.APIVersion,
			Kind:       operator.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      wsData.Name,
			Namespace: "default",
			Labels: map[string]string{
				operator.LabelWorkspaceID: wsData.ID.String(),
				operator.LabelUserID:      wsData.UserID.String(),
			},
		},
		Spec: operator.WorkspaceSpec{
			UserID:  wsData.UserID.String(),
			Image:   wsData.Image,
			RepoURL: wsData.RepoURL,
			Branch:  wsData.Branch,
			Resources: struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
				Disk   string `json:"disk"`
				GPU    int    `json:"gpu,omitempty"`
			}{
				CPU:    wsData.Resources.CPU,
				Memory: wsData.Resources.Memory,
				Disk:   wsData.Resources.Disk,
				GPU:    wsData.Resources.GPU,
			},
			Ingress: struct {
				Enabled bool               `json:"enabled"`
				Host    string             `json:"host,omitempty"`
				Ports   []operator.PortRule `json:"ports,omitempty"`
			}{
				Enabled: true,
				Host:    m.cfg.IngressHost,
				Ports:   p,
			},
			Storage: struct {
				Size         string `json:"size"`
				StorageClass string `json:"storageClass,omitempty"`
			}{
				Size: wsData.Resources.Disk,
			},
		},
	}

	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(wsOperator)
	if err != nil {
		return fmt.Errorf("failed to convert to unstructured: %w", err)
	}

	u := &unstructured.Unstructured{Object: unstructuredObj}
	_, err = m.dynamic.Resource(operator.SchemeGroupVersion.WithResource(operator.Plural)).
		Namespace("default").
		Create(ctx, u, metav1.CreateOptions{})
	if err != nil {
		slog.Error("CRD creation failed", "error", err, "workspace_id", event.WorkspaceID)
		return fmt.Errorf("failed to create workspace CRD: %w", err)
	}

	slog.Info("workspace CRD created", "workspace_id", event.WorkspaceID, "name", wsData.Name)

	m.publishStatus(ctx, event.WorkspaceID, event.UserID, "building")

	go m.watchCRDStatus(ctx, event.WorkspaceID, event.UserID, wsData.Name)
	return nil
}

func (m *Manager) publishStatus(ctx context.Context, workspaceID, userID uuid.UUID, status string) {
	payload, _ := json.Marshal(workspace.StatusUpdatePayload{Status: status})
	event := workspace.Event{
		ID:          uuid.New(),
		Type:        workspace.EventWorkspaceStatusUpdate,
		WorkspaceID: workspaceID,
		UserID:      userID,
		Payload:     payload,
		Timestamp:   time.Now().UTC(),
	}
	if err := m.publisher.Publish(ctx, event); err != nil {
		slog.Warn("failed to publish status update", "error", err, "workspace_id", workspaceID)
	}
}

func (m *Manager) watchCRDStatus(ctx context.Context, workspaceID, userID uuid.UUID, name string) {
	resource := m.dynamic.Resource(operator.SchemeGroupVersion.WithResource(operator.Plural))
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		u, err := resource.Namespace("default").Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			slog.Warn("failed to get CRD for status watch", "error", err, "workspace_id", workspaceID)
			continue
		}

		status, found, err := unstructured.NestedString(u.Object, "status", "phase")
		if err != nil || !found {
			continue
		}

		slog.Info("CRD status update", "workspace_id", workspaceID, "phase", status)

		m.publishStatus(ctx, workspaceID, userID, status)

		if status == "running" || status == "error" || status == "stopped" {
			return
		}
	}
}

func (m *Manager) handleStart(ctx context.Context, event workspace.Event) error {
	slog.Info("workspace start event", "workspace_id", event.WorkspaceID)

	restoreEvent := workspace.Event{
		ID:          event.ID,
		Type:        workspace.EventWorkspaceSnapshot,
		WorkspaceID: event.WorkspaceID,
		UserID:      event.UserID,
		Payload:     json.RawMessage(`{"action":"restore"}`),
		Timestamp:   time.Now().UTC(),
	}
	return m.publisher.Publish(ctx, restoreEvent)
}

func (m *Manager) handleStop(ctx context.Context, event workspace.Event) error {
	slog.Info("workspace stop event", "workspace_id", event.WorkspaceID)

	if m.cfg.SnapshotEnabled {
		snapEvent := workspace.Event{
			ID:          event.ID,
			Type:        workspace.EventWorkspaceSnapshot,
			WorkspaceID: event.WorkspaceID,
			UserID:      event.UserID,
			Payload:     json.RawMessage(`{"action":"backup"}`),
			Timestamp:   time.Now().UTC(),
		}
		return m.publisher.Publish(ctx, snapEvent)
	}

	slog.Info("stopping workspace pod", "workspace_id", event.WorkspaceID)
	wsID := event.WorkspaceID.String()
	err := m.k8s.CoreV1().Pods(wsID).Delete(ctx, wsID, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		slog.Warn("failed to delete pod", "workspace_id", wsID, "error", err)
	}

	return nil
}

func (m *Manager) handleDelete(ctx context.Context, event workspace.Event) error {
	slog.Info("workspace delete event", "workspace_id", event.WorkspaceID)

	ns := event.WorkspaceID.String()
	if err := m.k8s.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{}); err != nil {
		slog.Warn("failed to delete namespace", "namespace", ns, "error", err)
	}
	return nil
}

func (m *Manager) handleSnapshot(ctx context.Context, event workspace.Event) error {
	slog.Info("workspace snapshot event", "workspace_id", event.WorkspaceID)

	var payload struct {
		Action     string `json:"action"`
		SnapshotID string `json:"snapshot_id,omitempty"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	wsID := event.WorkspaceID.String()

	switch payload.Action {
	case "backup":
		if err := m.snapshots.Backup(ctx, wsID, wsID, wsID); err != nil {
			slog.Error("snapshot backup failed", "workspace_id", wsID, "error", err)
			return err
		}
	case "restore":
		if err := m.snapshots.Restore(ctx, wsID, wsID, wsID); err != nil {
			slog.Warn("snapshot restore failed (may be first start)", "workspace_id", wsID, "error", err)
		}
	case "schedule":
		var schedulePayload workspace.ScheduleSnapshotPayload
		if err := json.Unmarshal(event.Payload, &schedulePayload); err == nil {
			for _, id := range schedulePayload.WorkspaceIDs {
				backupEvent := workspace.Event{
					ID:          event.ID,
					Type:        workspace.EventWorkspaceSnapshot,
					WorkspaceID: event.WorkspaceID,
					Payload:     json.RawMessage(`{"action":"backup"}`),
					Timestamp:   time.Now().UTC(),
				}
				if id != "" {
					backupEvent.WorkspaceID = uuid.MustParse(id)
				}
				m.publisher.Publish(ctx, backupEvent)
			}
		}
	}

	return nil
}

func (m *Manager) handleTimeout(ctx context.Context, event workspace.Event) error {
	slog.Info("workspace timeout event, stopping workspace", "workspace_id", event.WorkspaceID)

	wsID := event.WorkspaceID.String()

	pod, err := m.k8s.CoreV1().Pods(wsID).Get(ctx, wsID, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		slog.Warn("failed to get pod for timeout", "workspace_id", wsID, "error", err)
		return nil
	}

	if pod.Status.Phase == corev1.PodRunning {
		err = m.k8s.CoreV1().Pods(wsID).Delete(ctx, wsID, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			slog.Warn("failed to delete idle pod", "workspace_id", wsID, "error", err)
		}
		slog.Info("stopped idle workspace", "workspace_id", wsID)
	}

	return nil
}

func (m *Manager) handlePrebuild(ctx context.Context, event workspace.Event) error {
	slog.Info("prebuild event", "prebuild_id", event.WorkspaceID)
	return nil
}



