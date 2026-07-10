package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nimbuscore/apps/api/internal/store"
	"github.com/nimbuscore/pkg/api"
	"github.com/nimbuscore/pkg/auth"
)

type upsertQuotaInput struct {
	MaxWorkspaces int    `json:"max_workspaces"`
	MaxCPU        string `json:"max_cpu"`
	MaxMemory     string `json:"max_memory"`
	MaxDisk       string `json:"max_disk"`
	MaxGPU        int    `json:"max_gpu,omitempty"`
}

func (h *Handler) GetQuota(w http.ResponseWriter, r *http.Request) {
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}

	q, err := store.GetQuota(r.Context(), h.db, teamID)
	if err != nil {
		http.Error(w, "quota not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

func (h *Handler) UpsertQuota(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}

	var input upsertQuotaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	q := api.Quota{
		TeamID:        teamID,
		MaxWorkspaces: input.MaxWorkspaces,
		MaxCPU:        input.MaxCPU,
		MaxMemory:     input.MaxMemory,
		MaxDisk:       input.MaxDisk,
		MaxGPU:        input.MaxGPU,
	}

	if err := store.UpsertQuota(r.Context(), h.db, &q); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

func (h *Handler) DeleteQuota(w http.ResponseWriter, r *http.Request) {
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}

	if err := store.DeleteQuota(r.Context(), h.db, teamID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CheckQuota(w http.ResponseWriter, r *http.Request) {
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}

	q, err := store.GetQuota(r.Context(), h.db, teamID)
	if err != nil {
		http.Error(w, "quota not found", http.StatusNotFound)
		return
	}

	available := q.MaxWorkspaces == 0 || q.UsedWorkspaces < q.MaxWorkspaces
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"available":      available,
		"used":           q.UsedWorkspaces,
		"max":            q.MaxWorkspaces,
		"used_cpu":       q.UsedCPU,
		"max_cpu":        q.MaxCPU,
		"used_memory":    q.UsedMemory,
		"max_memory":     q.MaxMemory,
		"used_disk":      q.UsedDisk,
		"max_disk":       q.MaxDisk,
	})
}
