package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nimbuscore/apps/api/internal/store"
	"github.com/nimbuscore/pkg/api"
	"github.com/nimbuscore/pkg/auth"
	"github.com/nimbuscore/pkg/workspace"
)

type createSnapshotInput struct {
	Action string `json:"action"`
}

func (h *Handler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceId"))
	if err != nil {
		http.Error(w, "invalid workspace id", http.StatusBadRequest)
		return
	}

	snaps, err := store.ListSnapshots(r.Context(), h.db, wsID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snaps)
}

func (h *Handler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceId"))
	if err != nil {
		http.Error(w, "invalid workspace id", http.StatusBadRequest)
		return
	}

	ws, err := store.GetWorkspace(r.Context(), h.db, wsID)
	if err != nil {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	if ws.UserID != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	snap := api.Snapshot{
		ID:          uuid.New(),
		WorkspaceID: wsID,
		Status:      api.SnapshotStatusPending,
		StartedAt:   time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	}

	if err := store.CreateSnapshot(r.Context(), h.db, &snap); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	payload, _ := json.Marshal(workspace.SnapshotEventPayload{
		Action:     "backup",
		SnapshotID: snap.ID.String(),
	})
	event := workspace.Event{
		ID:          uuid.New(),
		Type:        workspace.EventWorkspaceSnapshot,
		WorkspaceID: wsID,
		UserID:      claims.UserID,
		Payload:     payload,
		Timestamp:   time.Now().UTC(),
	}

	if h.rmq != nil {
		pub := workspace.NewPublisher(h.rmq.Channel(), "nimbuscore.workspace")
		pub.Publish(r.Context(), event)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(snap)
}

func (h *Handler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid snapshot id", http.StatusBadRequest)
		return
	}

	snap, err := store.GetSnapshot(r.Context(), h.db, id)
	if err != nil {
		http.Error(w, "snapshot not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snap)
}

func (h *Handler) RestoreSnapshot(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	snapID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid snapshot id", http.StatusBadRequest)
		return
	}

	snap, err := store.GetSnapshot(r.Context(), h.db, snapID)
	if err != nil {
		http.Error(w, "snapshot not found", http.StatusNotFound)
		return
	}

	ws, err := store.GetWorkspace(r.Context(), h.db, snap.WorkspaceID)
	if err != nil {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	if ws.UserID != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	payload, _ := json.Marshal(workspace.SnapshotEventPayload{
		Action:     "restore",
		SnapshotID: snapID.String(),
	})
	event := workspace.Event{
		ID:          uuid.New(),
		Type:        workspace.EventWorkspaceSnapshot,
		WorkspaceID: snap.WorkspaceID,
		UserID:      claims.UserID,
		Payload:     payload,
		Timestamp:   time.Now().UTC(),
	}

	if h.rmq != nil {
		pub := workspace.NewPublisher(h.rmq.Channel(), "nimbuscore.workspace")
		pub.Publish(r.Context(), event)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "restore initiated"})
}
