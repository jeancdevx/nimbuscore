package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/nimbuscore/apps/api/internal/handler"
	"github.com/nimbuscore/apps/api/internal/middleware"
	"github.com/nimbuscore/pkg/auth"
)

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	jwtIssuer := auth.NewJWTIssuer(s.cfg.Auth.JWTSecret, time.Duration(s.cfg.Auth.SessionTTL)*time.Hour)
	jwtMiddleware := auth.Middleware(jwtIssuer)

	h := handler.New(s.cfg, s.db, s.rdb, s.rmq, jwtIssuer)

	r.Get("/health", h.Health)

	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", h.AuthLogin)
		r.Get("/callback", h.AuthCallback)
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(jwtMiddleware)

		r.Route("/users", func(r chi.Router) {
			r.Get("/", h.ListUsers)
			r.Get("/me", h.GetCurrentUser)
			r.Get("/{id}", h.GetUser)
		})

		r.Route("/teams", func(r chi.Router) {
			r.Get("/", h.ListTeams)
			r.Post("/", h.CreateTeam)
			r.Get("/{id}", h.GetTeam)
			r.Patch("/{id}", h.UpdateTeam)
			r.Delete("/{id}", h.DeleteTeam)
		})

		r.Route("/workspaces", func(r chi.Router) {
			r.Get("/", h.ListWorkspaces)
			r.Post("/", h.CreateWorkspace)
			r.Get("/{id}", h.GetWorkspace)
			r.Delete("/{id}", h.DeleteWorkspace)
			r.Post("/{id}/start", h.StartWorkspace)
			r.Post("/{id}/stop", h.StopWorkspace)
		})
	})

	return r
}
