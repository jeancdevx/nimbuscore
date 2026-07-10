package snapshot

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nimbuscore/pkg/workspace"
)

type Scheduler struct {
	svc      *Service
	pub      *workspace.Publisher
	interval time.Duration
	k8s      kubernetes.Interface
}

func NewScheduler(svc *Service, pub *workspace.Publisher, schedule string, k8s kubernetes.Interface) *Scheduler {
	d, err := time.ParseDuration(schedule)
	if err != nil {
		d = 30 * time.Minute
	}
	return &Scheduler{svc: svc, pub: pub, interval: d, k8s: k8s}
}

func (s *Scheduler) Start(ctx context.Context) {
	slog.Info("snapshot scheduler starting", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.runScheduledSnapshots(ctx)
		case <-ctx.Done():
			slog.Info("snapshot scheduler stopped")
			return
		}
	}
}

func (s *Scheduler) runScheduledSnapshots(ctx context.Context) {
	slog.Info("running scheduled snapshots for active workspaces")

	namespaces, err := s.k8s.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: "nimbuscore.io/workspace-id",
	})
	if err != nil {
		slog.Error("failed to list workspace namespaces", "error", err)
		return
	}

	for _, ns := range namespaces.Items {
		wsID, ok := ns.Labels["nimbuscore.io/workspace-id"]
		if !ok || wsID == "" {
			continue
		}

		pods, err := s.k8s.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{
			LabelSelector: "nimbuscore.io/workspace-id=" + wsID,
		})
		if err != nil {
			continue
		}

		hasRunning := false
		for _, pod := range pods.Items {
			if pod.Status.Phase == "Running" {
				hasRunning = true
				break
			}
		}

		if !hasRunning {
			continue
		}

		parsedID, err := uuid.Parse(wsID)
		if err != nil {
			continue
		}

		slog.Info("scheduled snapshot for workspace", "workspace_id", wsID)

		payload, _ := json.Marshal(workspace.SnapshotEventPayload{
			Action: "backup",
		})
		event := workspace.Event{
			ID:          uuid.New(),
			Type:        workspace.EventWorkspaceSnapshot,
			WorkspaceID: parsedID,
			Payload:     payload,
			Timestamp:   time.Now().UTC(),
		}

		if err := s.pub.Publish(ctx, event); err != nil {
			slog.Error("failed to publish snapshot event", "workspace_id", wsID, "error", err)
		}
	}
}
