package watcher

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nimbuscore/pkg/workspace"
)

type IdleWatcher struct {
	k8s         kubernetes.Interface
	pub         *workspace.Publisher
	idleTimeout int
}

func NewIdleWatcher(k8s kubernetes.Interface, pub *workspace.Publisher, idleTimeout int) *IdleWatcher {
	return &IdleWatcher{k8s: k8s, pub: pub, idleTimeout: idleTimeout}
}

func (w *IdleWatcher) Start(ctx context.Context) {
	slog.Info("idle watcher starting", "timeout_minutes", w.idleTimeout)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.checkIdleWorkspaces(ctx)
		case <-ctx.Done():
			slog.Info("idle watcher stopped")
			return
		}
	}
}

func (w *IdleWatcher) checkIdleWorkspaces(ctx context.Context) {
	namespaces, err := w.k8s.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: "nimbuscore.io/workspace-id",
	})
	if err != nil {
		slog.Error("failed to list namespaces", "error", err)
		return
	}

	for _, ns := range namespaces.Items {
		wsID, ok := ns.Labels["nimbuscore.io/workspace-id"]
		if !ok {
			continue
		}

		pods, err := w.k8s.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{
			LabelSelector: "nimbuscore.io/workspace-id=" + wsID,
		})
		if err != nil {
			continue
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}

			hasActive := false
			for _, c := range pod.Status.ContainerStatuses {
				if c.State.Running != nil {
					hasActive = true
					break
				}
			}
			if !hasActive {
				continue
			}

			runningDuration := time.Since(pod.CreationTimestamp.Time)
			if runningDuration > time.Duration(w.idleTimeout)*time.Minute {
				slog.Info("workspace idle timeout reached",
					"workspace_id", wsID,
					"running_duration", runningDuration,
				)

				event := workspace.Event{
					ID:          uuid.New(),
					Type:        workspace.EventWorkspaceTimeout,
					WorkspaceID: uuid.MustParse(wsID),
					Timestamp:   time.Now().UTC(),
				}

				if err := w.pub.Publish(ctx, event); err != nil {
					slog.Error("failed to publish timeout event", "workspace_id", wsID, "error", err)
				}
			}
		}
	}
}
