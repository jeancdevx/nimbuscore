package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nimbuscore/apps/api/internal/store"
	"github.com/nimbuscore/pkg/api"
	"github.com/nimbuscore/pkg/auth"
)

func (h *Handler) AuthLogin(w http.ResponseWriter, r *http.Request) {
	provider, err := auth.NewProvider(
		r.Context(),
		h.cfg.Auth.Issuer,
		h.cfg.Auth.ClientID,
		h.cfg.Auth.ClientSecret,
		h.cfg.Auth.RedirectURL,
	)
	if err != nil {
		slog.Error("failed to create OIDC provider", "error", err)
		http.Error(w, "authentication unavailable", http.StatusInternalServerError)
		return
	}

	state := uuid.New().String()
	http.Redirect(w, r, provider.AuthURL(state), http.StatusTemporaryRedirect)
}

func (h *Handler) AuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	provider, err := auth.NewProvider(
		r.Context(),
		h.cfg.Auth.Issuer,
		h.cfg.Auth.ClientID,
		h.cfg.Auth.ClientSecret,
		h.cfg.Auth.RedirectURL,
	)
	if err != nil {
		slog.Error("failed to create OIDC provider", "error", err)
		http.Error(w, "authentication unavailable", http.StatusInternalServerError)
		return
	}

	oauthToken, err := provider.Exchange(r.Context(), code)
	if err != nil {
		slog.Error("failed to exchange auth code", "error", err)
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	rawIDToken, ok := oauthToken.Extra("id_token").(string)
	if !ok {
		http.Error(w, "missing id_token", http.StatusInternalServerError)
		return
	}

	idToken, err := provider.VerifyIDToken(r.Context(), rawIDToken)
	if err != nil {
		slog.Error("failed to verify id_token", "error", err)
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	var claims struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		slog.Error("failed to parse id_token claims", "error", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	user := api.User{
		ID:         uuid.New(),
		Email:      claims.Email,
		Name:       claims.Name,
		Provider:   "dex",
		ProviderID: idToken.Subject,
		AvatarURL:  claims.Picture,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := store.UpsertUser(r.Context(), h.db, &user); err != nil {
		slog.Error("failed to upsert user", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	role := string(api.RoleUser)
	if string(user.Role) != "" {
		role = string(user.Role)
	}

	jwt, err := h.jwtAuth.Issue(user.ID, user.Email, user.Name, role, user.AvatarURL)
	if err != nil {
		slog.Error("failed to issue JWT", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":   jwt,
		"user_id": user.ID.String(),
	})
}
