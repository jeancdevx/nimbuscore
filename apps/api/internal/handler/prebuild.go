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

type createPrebuildInput struct {
	Name            string `json:"name"`
	Image           string `json:"image"`
	RepoURL         string `json:"repo_url"`
	Branch          string `json:"branch"`
	DevcontainerPath string `json:"devcontainer_path,omitempty"`
	OutputImage     string `json:"output_image"`
}

func (h *Handler) ListPrebuilds(w http.ResponseWriter, r *http.Request) {
	prebuilds, err := store.ListPrebuilds(r.Context(), h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prebuilds)
}

func (h *Handler) CreatePrebuild(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input createPrebuildInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" || input.RepoURL == "" || input.OutputImage == "" {
		http.Error(w, "name, repo_url, and output_image are required", http.StatusBadRequest)
		return
	}

	pb := api.Prebuild{
		ID:              uuid.New(),
		UserID:          claims.UserID,
		Name:            input.Name,
		Image:           input.Image,
		RepoURL:         input.RepoURL,
		Branch:          input.Branch,
		DevcontainerPath: input.DevcontainerPath,
		Status:          api.PrebuildStatusPending,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := store.CreatePrebuild(r.Context(), h.db, &pb); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	event := workspace.Event{
		ID:        uuid.New(),
		Type:      workspace.EventWorkspacePrebuild,
		WorkspaceID: pb.ID,
		UserID:    claims.UserID,
		Timestamp: time.Now().UTC(),
	}

	if h.rmq != nil {
		pub := workspace.NewPublisher(h.rmq.Channel(), "nimbuscore.workspace")
		pub.Publish(r.Context(), event)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pb)
}

func (h *Handler) GetPrebuild(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid prebuild id", http.StatusBadRequest)
		return
	}

	pb, err := store.GetPrebuild(r.Context(), h.db, id)
	if err != nil {
		http.Error(w, "prebuild not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pb)
}

func (h *Handler) DeletePrebuild(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid prebuild id", http.StatusBadRequest)
		return
	}

	if err := store.DeletePrebuild(r.Context(), h.db, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
