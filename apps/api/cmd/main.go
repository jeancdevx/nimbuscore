package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/nimbuscore/apps/api/internal/cache"
	"github.com/nimbuscore/apps/api/internal/handler"
	"github.com/nimbuscore/apps/api/internal/queue"
	"github.com/nimbuscore/apps/api/internal/server"
	"github.com/nimbuscore/apps/api/internal/store"
	"github.com/nimbuscore/pkg/api"
	"github.com/nimbuscore/pkg/workspace"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := loadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := store.NewPool(ctx, cfg.Database.URL, cfg.Database.MaxConns)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	rdb := cache.NewRedis(cfg.Redis.URL, cfg.Redis.Password, cfg.Redis.DB)

	var pingErr error
	for range 5 {
		pingErr = rdb.Ping(ctx).Err()
		if pingErr == nil {
			break
		}
		slog.Error("failed to connect to redis", "error", pingErr.Error())
		time.Sleep(2 * time.Second)
	}
	if pingErr != nil {
		slog.Error("redis connection failed after retries", "error", pingErr.Error())
		os.Exit(1)
	}

	rmq, err := queue.NewRabbitMQ(cfg.RabbitMQ.URL)
	if err != nil {
		slog.Error("failed to connect to rabbitmq", "error", err)
		os.Exit(1)
	}
	defer rmq.Close()

	srv := server.New(cfg, db, rdb, rmq)

	go startStatusConsumer(context.Background(), rmq, rdb, db)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Handler:      srv.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server starting", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdown
	slog.Info("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

func loadConfig() api.APIConfig {
	return api.APIConfig{
		Host:        getEnv("HOST", "0.0.0.0"),
		Port:        getEnv("PORT", "8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		IngressHost: getEnv("INGRESS_HOST", "localhost"),
		Database: api.DatabaseConfig{
			URL:      getEnv("DATABASE_URL", "postgres://nimbuscore:nimbuscore@localhost:5432/nimbuscore?sslmode=disable"),
			MaxConns: 25,
			MinConns: 5,
		},
		Redis: api.RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		RabbitMQ: api.RabbitMQConfig{
			URL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672"),
		},
		Auth: api.AuthConfig{
			Issuer:         getEnv("OIDC_ISSUER", "http://localhost:5556/dex"),
			ExternalIssuer: getEnv("OIDC_EXTERNAL_ISSUER", "http://localhost:5556/dex"),
			ClientID:       getEnv("OIDC_CLIENT_ID", "nimbuscore"),
			ClientSecret:   getEnv("OIDC_CLIENT_SECRET", ""),
			RedirectURL:    getEnv("OIDC_REDIRECT_URL", "http://localhost:8080/auth/callback"),
			JWTSecret:      getEnv("JWT_SECRET", "change-me-in-production"),
			SessionTTL:     24,
		},
	}
}

func startStatusConsumer(ctx context.Context, rmq *queue.RabbitMQ, rdb *redis.Client, db *pgxpool.Pool) {
	consumerCh, err := rmq.NewChannel()
	if err != nil {
		slog.Error("failed to create consumer channel", "error", err)
		return
	}
	consumer := workspace.NewConsumer(consumerCh, "nimbuscore.workspace", "api-status-updater")
	consumer.Handle(workspace.EventWorkspaceStatusUpdate, func(ctx context.Context, event workspace.Event) error {
		var payload workspace.StatusUpdatePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			slog.Error("failed to unmarshal status update payload", "error", err)
			return nil
		}

		status := api.WorkspaceStatus(payload.Status)
		if err := store.UpdateWorkspaceStatus(ctx, db, event.WorkspaceID, status); err != nil {
			slog.Error("failed to update workspace status", "error", err, "workspace_id", event.WorkspaceID)
			return err
		}

		handler.PublishWorkspaceStatus(ctx, rdb, event.WorkspaceID.String(), string(status))

		slog.Info("workspace status updated", "workspace_id", event.WorkspaceID, "status", status)
		return nil
	})

	if err := consumer.Start(ctx); err != nil {
		slog.Error("status consumer exited", "error", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
