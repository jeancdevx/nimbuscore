package snapshot

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nimbuscore/pkg/workspace"
)

type Scheduler struct {
	svc      *Service
	pub      *workspace.Publisher
	interval time.Duration
}

func NewScheduler(svc *Service, pub *workspace.Publisher, schedule string) *Scheduler {
	d, err := time.ParseDuration(schedule)
	if err != nil {
		d = 30 * time.Minute
	}
	return &Scheduler{svc: svc, pub: pub, interval: d}
}

func (s *Scheduler) Start(ctx context.Context) {
	slog.Info("snapshot scheduler starting", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			slog.Info("triggering periodic workspace snapshots")

			event := workspace.Event{
				ID:        uuid.New(),
				Type:      workspace.EventWorkspaceSnapshot,
				Timestamp: time.Now().UTC(),
			}

			if err := s.pub.Publish(ctx, event); err != nil {
				slog.Error("failed to publish snapshot event", "error", err)
			}

		case <-ctx.Done():
			slog.Info("snapshot scheduler stopped")
			return
		}
	}
}
