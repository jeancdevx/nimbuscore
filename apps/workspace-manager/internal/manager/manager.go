package manager

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nimbuscore/apps/workspace-manager/internal/snapshot"
	"github.com/nimbuscore/pkg/workspace"
)

type Manager struct {
	cfg       Config
	k8s       kubernetes.Interface
	consumer  *workspace.Consumer
	publisher *workspace.Publisher
	snapshots *snapshot.Service
}

type Config struct {
	Ingress         workspace.IngressConfig
	SnapshotEnabled bool
	IdleTimeout     int
}

func New(
	cfg Config,
	k8s kubernetes.Interface,
	rmqCh *amqp.Channel,
	snapshots *snapshot.Service,
	publisher *workspace.Publisher,
) *Manager {
	consumer := workspace.NewConsumer(rmqCh, "nimbuscore.workspace", "workspace-manager")

	mgr := &Manager{
		cfg:       cfg,
		k8s:       k8s,
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

	return mgr
}

func (m *Manager) Start(ctx context.Context) error {
	slog.Info("workspace manager listening for events")
	return m.consumer.Start(ctx)
}

func (m *Manager) handleCreate(ctx context.Context, event workspace.Event) error {
	slog.Info("workspace create event", "workspace_id", event.WorkspaceID)
	return nil
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
		Action string `json:"action"`
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
	}

	return nil
}

func (m *Manager) handleTimeout(ctx context.Context, event workspace.Event) error {
	slog.Info("workspace timeout event", "workspace_id", event.WorkspaceID)
	return nil
}
