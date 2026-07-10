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

type createWorkspaceInput struct {
	Name      string                 `json:"name"`
	Image     string                 `json:"image"`
	RepoURL   string                 `json:"repo_url"`
	Branch    string                 `json:"branch"`
	TeamID    *uuid.UUID             `json:"team_id,omitempty"`
	Ports     []api.PortMapping      `json:"ports,omitempty"`
	Resources api.WorkspaceResources `json:"resources"`
}

func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	workspaces, err := store.ListWorkspacesByUser(r.Context(), h.db, claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workspaces)
}

func (h *Handler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input createWorkspaceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if input.TeamID != nil {
		q, err := store.GetQuota(r.Context(), h.db, *input.TeamID)
		if err == nil && !workspace.CheckQuota(*q, input.Resources) {
			http.Error(w, "team quota exceeded", http.StatusConflict)
			return
		}
	}

	ws := workspace.NewWorkspace(workspace.CreateWorkspaceInput{
		Name:   input.Name,
		UserID: claims.UserID,
		TeamID: input.TeamID,
		Image:  input.Image,
		Resources: api.WorkspaceResources{
			CPU:    input.Resources.CPU,
			Memory: input.Resources.Memory,
			Disk:   input.Resources.Disk,
			GPU:    input.Resources.GPU,
		},
		RepoURL: input.RepoURL,
		Branch:  input.Branch,
		Ports:   input.Ports,
	})

	if err := store.CreateWorkspace(r.Context(), h.db, ws); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	event := workspace.Event{
		ID:          uuid.New(),
		Type:        workspace.EventWorkspaceCreate,
		WorkspaceID: ws.ID,
		UserID:      claims.UserID,
		Timestamp:   time.Now().UTC(),
	}
	if h.rmq != nil {
		pub := workspace.NewPublisher(h.rmq.Channel(), "nimbuscore.workspace")
		pub.Publish(r.Context(), event)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ws)
}

func (h *Handler) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid workspace id", http.StatusBadRequest)
		return
	}

	ws, err := store.GetWorkspace(r.Context(), h.db, id)
	if err != nil {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ws)
}

func (h *Handler) DeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid workspace id", http.StatusBadRequest)
		return
	}

	ws, err := store.GetWorkspace(r.Context(), h.db, id)
	if err != nil {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	if ws.UserID != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if !workspace.CanTransition(ws.Status, api.WorkspaceStatusDeleted) {
		http.Error(w, "cannot delete workspace in current state", http.StatusConflict)
		return
	}

	if err := store.DeleteWorkspace(r.Context(), h.db, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	event := workspace.Event{
		ID:          uuid.New(),
		Type:        workspace.EventWorkspaceDelete,
		WorkspaceID: id,
		UserID:      claims.UserID,
		Timestamp:   time.Now().UTC(),
	}
	if h.rmq != nil {
		pub := workspace.NewPublisher(h.rmq.Channel(), "nimbuscore.workspace")
		pub.Publish(r.Context(), event)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) StartWorkspace(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid workspace id", http.StatusBadRequest)
		return
	}

	ws, err := store.GetWorkspace(r.Context(), h.db, id)
	if err != nil {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	if ws.UserID != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if !workspace.CanTransition(ws.Status, api.WorkspaceStatusBuilding) {
		http.Error(w, "cannot start workspace in current state", http.StatusConflict)
		return
	}

	prev := ws.Status
	ws.Status = api.WorkspaceStatusBuilding
	if err := store.UpdateWorkspaceStatus(r.Context(), h.db, id, ws.Status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	event := workspace.Event{
		ID:          uuid.New(),
		Type:        workspace.EventWorkspaceStart,
		WorkspaceID: id,
		UserID:      claims.UserID,
		Timestamp:   time.Now().UTC(),
	}
	if h.rmq != nil {
		pub := workspace.NewPublisher(h.rmq.Channel(), "nimbuscore.workspace")
		pub.Publish(r.Context(), event)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message":    "workspace starting",
		"prev_state": prev,
		"new_state":  ws.Status,
	})
}

func (h *Handler) StopWorkspace(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid workspace id", http.StatusBadRequest)
		return
	}

	ws, err := store.GetWorkspace(r.Context(), h.db, id)
	if err != nil {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	if ws.UserID != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if !workspace.CanTransition(ws.Status, api.WorkspaceStatusStopping) {
		http.Error(w, "cannot stop workspace in current state", http.StatusConflict)
		return
	}

	prev := ws.Status
	ws.Status = api.WorkspaceStatusStopping
	if err := store.UpdateWorkspaceStatus(r.Context(), h.db, id, ws.Status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	event := workspace.Event{
		ID:          uuid.New(),
		Type:        workspace.EventWorkspaceStop,
		WorkspaceID: id,
		UserID:      claims.UserID,
		Timestamp:   time.Now().UTC(),
	}
	if h.rmq != nil {
		pub := workspace.NewPublisher(h.rmq.Channel(), "nimbuscore.workspace")
		pub.Publish(r.Context(), event)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message":    "workspace stopping",
		"prev_state": prev,
		"new_state":  ws.Status,
	})
}

func (h *Handler) HeartbeatWorkspace(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid workspace id", http.StatusBadRequest)
		return
	}

	ws, err := store.GetWorkspace(r.Context(), h.db, id)
	if err != nil {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	if ws.UserID != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	store.UpdateWorkspaceLastActivity(r.Context(), h.db, id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
