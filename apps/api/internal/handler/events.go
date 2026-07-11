package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"
)

const workspaceStatusChannel = "workspace:status"

func PublishWorkspaceStatus(ctx context.Context, rdb *redis.Client, workspaceID, status string) {
	data, _ := json.Marshal(map[string]string{
		"id":     workspaceID,
		"status": status,
	})
	if err := rdb.Publish(ctx, workspaceStatusChannel, string(data)).Err(); err != nil {
		slog.Warn("failed to publish workspace status", "error", err)
	}
}

func (h *Handler) EventStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	pubsub := h.rdb.Subscribe(r.Context(), workspaceStatusChannel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "event: workspace.status\ndata: %s\n\n", msg.Payload)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
