package server

import (
	"expvar"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/nimbuscore/apps/api/internal/handler"
	"github.com/nimbuscore/apps/api/internal/middleware"
	"github.com/nimbuscore/pkg/api"
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
	auditMiddleware := middleware.AuditLogger(s.db)

	h := handler.New(s.cfg, s.db, s.rdb, s.rmq, jwtIssuer)

	r.Get("/health", h.Health)
	r.Get("/metrics", expvar.Handler().ServeHTTP)

	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", h.AuthLogin)
		r.Get("/callback", h.AuthCallback)
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(jwtMiddleware)
		r.Use(auditMiddleware)
		r.Use(middleware.RBACInjector)

		r.Get("/events", h.EventStream)

		r.Route("/users", func(r chi.Router) {
			r.Get("/", h.ListUsers)
			r.Get("/me", h.GetCurrentUser)
			r.Get("/{id}", h.GetUser)
			r.With(middleware.RequirePermission(api.PermissionManageSystem)).
				Patch("/{id}/role", h.UpdateUserRole)
		})

		r.Route("/teams", func(r chi.Router) {
			r.Get("/", h.ListTeams)
			r.With(middleware.RequirePermission(api.PermissionManageTeams)).
				Post("/", h.CreateTeam)
			r.Get("/{id}", h.GetTeam)
			r.With(middleware.RequirePermission(api.PermissionManageTeams)).
				Patch("/{id}", h.UpdateTeam)
			r.With(middleware.RequirePermission(api.PermissionManageTeams)).
				Delete("/{id}", h.DeleteTeam)

			r.Route("/{teamId}/quota", func(r chi.Router) {
				r.Get("/", h.GetQuota)
				r.With(middleware.RequirePermission(api.PermissionManageQuotas)).
					Put("/", h.UpsertQuota)
				r.With(middleware.RequirePermission(api.PermissionManageQuotas)).
					Delete("/", h.DeleteQuota)
				r.Get("/check", h.CheckQuota)
			})

			r.Route("/{teamId}/billing", func(r chi.Router) {
				r.With(middleware.RequirePermission(api.PermissionViewBilling)).
					Get("/", h.GetBilling)
			})
		})

		r.Route("/workspaces", func(r chi.Router) {
			r.Get("/", h.ListWorkspaces)
			r.Post("/", h.CreateWorkspace)
			r.Get("/{id}", h.GetWorkspace)
			r.Delete("/{id}", h.DeleteWorkspace)
			r.Post("/{id}/start", h.StartWorkspace)
			r.Post("/{id}/stop", h.StopWorkspace)
			r.Post("/{id}/heartbeat", h.HeartbeatWorkspace)

			r.Route("/{workspaceId}/snapshots", func(r chi.Router) {
				r.Get("/", h.ListSnapshots)
				r.Post("/", h.CreateSnapshot)
				r.Get("/{id}", h.GetSnapshot)
				r.Post("/{id}/restore", h.RestoreSnapshot)
			})
		})

		r.Route("/prebuilds", func(r chi.Router) {
			r.Get("/", h.ListPrebuilds)
			r.Post("/", h.CreatePrebuild)
			r.Get("/{id}", h.GetPrebuild)
			r.Delete("/{id}", h.DeletePrebuild)
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.RequirePermission(api.PermissionManageSystem))

			r.Route("/audit-log", func(r chi.Router) {
				r.With(middleware.RequirePermission(api.PermissionReadAuditLog)).
					Get("/", h.ListAuditLogs)
			})

			r.Route("/clusters", func(r chi.Router) {
				r.Get("/", h.ListClusters)
				r.Post("/", h.CreateCluster)
				r.Delete("/{id}", h.DeleteCluster)
			})
		})
	})

	return r
}
