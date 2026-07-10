package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nimbuscore/apps/api/internal/middleware"
	"github.com/nimbuscore/apps/api/internal/store"
	"github.com/nimbuscore/pkg/api"
	"github.com/nimbuscore/pkg/auth"
)

func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	entries, err := store.ListAuditLogs(r.Context(), h.db, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (h *Handler) GetBilling(w http.ResponseWriter, r *http.Request) {
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}

	since := time.Now().Add(-30 * 24 * time.Hour)
	until := time.Now()

	if s := r.URL.Query().Get("since"); s != "" {
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			since = parsed
		}
	}
	if u := r.URL.Query().Get("until"); u != "" {
		if parsed, err := time.Parse(time.RFC3339, u); err == nil {
			until = parsed
		}
	}

	records, err := store.GetBillingByTeam(r.Context(), h.db, teamID, since, until)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalCost := 0.0
	totalMinutes := 0
	for _, rec := range records {
		totalCost += rec.Cost
		totalMinutes += rec.DurationMin
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"records":      records,
		"total_cost":   totalCost,
		"total_minutes": totalMinutes,
		"since":        since,
		"until":        until,
	})
}

func (h *Handler) ListClusters(w http.ResponseWriter, r *http.Request) {
	clusters, err := store.ListClusters(r.Context(), h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clusters)
}

func (h *Handler) CreateCluster(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string   `json:"name"`
		APIEndpoint string   `json:"api_endpoint"`
		Region      string   `json:"region"`
		Provider    string   `json:"provider"`
		Labels      []string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" || input.APIEndpoint == "" {
		http.Error(w, "name and api_endpoint are required", http.StatusBadRequest)
		return
	}

	cluster := api.Cluster{
		ID:          uuid.New(),
		Name:        input.Name,
		APIEndpoint: input.APIEndpoint,
		Region:      input.Region,
		Provider:    input.Provider,
		Enabled:     true,
		Labels:      input.Labels,
	}

	if err := store.CreateCluster(r.Context(), h.db, &cluster); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cluster)
}

func (h *Handler) DeleteCluster(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid cluster id", http.StatusBadRequest)
		return
	}

	if err := store.DeleteCluster(r.Context(), h.db, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var input struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	role := api.Role(input.Role)
	switch role {
	case api.RoleAdmin, api.RoleTeamAdmin, api.RoleUser:
	default:
		http.Error(w, "invalid role, must be admin, team_admin, or user", http.StatusBadRequest)
		return
	}

	if err := store.UpdateUserRole(r.Context(), h.db, userID, role); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.LogAuditEntry(r.Context(), api.AuditUserRoleChange,
		userID.String(), "user", input.Role)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "role updated"})
}
