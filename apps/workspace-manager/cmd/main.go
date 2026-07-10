package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/nimbuscore/apps/workspace-manager/internal/manager"
	"github.com/nimbuscore/apps/workspace-manager/internal/snapshot"
	"github.com/nimbuscore/apps/workspace-manager/internal/watcher"
	"github.com/nimbuscore/pkg/workspace"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := loadConfig()

	k8sClient, err := newK8sClient()
	if err != nil {
		slog.Error("failed to create k8s client", "error", err)
		os.Exit(1)
	}

	rmqConn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		slog.Error("failed to connect to rabbitmq", "error", err)
		os.Exit(1)
	}
	defer rmqConn.Close()

	rmqCh, err := rmqConn.Channel()
	if err != nil {
		slog.Error("failed to open rabbitmq channel", "error", err)
		os.Exit(1)
	}
	defer rmqCh.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	snapshotSvc := snapshot.New(cfg.Snapshot, cfg.ResticImage)
	eventPublisher := workspace.NewPublisher(rmqCh, "nimbuscore.workspace")

	mgrCfg := manager.Config{
		Ingress:         cfg.Ingress,
		SnapshotEnabled: cfg.Snapshot.Enabled,
		IdleTimeout:     cfg.IdleTimeout,
	}
	mgr := manager.New(mgrCfg, k8sClient, rmqCh, snapshotSvc, eventPublisher)

	idlWatcher := watcher.NewIdleWatcher(k8sClient, eventPublisher, cfg.IdleTimeout)
	go idlWatcher.Start(ctx)

	snapshotScheduler := snapshot.NewScheduler(snapshotSvc, eventPublisher, cfg.SnapshotSchedule)
	go snapshotScheduler.Start(ctx)

	slog.Info("workspace-manager starting", "rabbitmq", cfg.RabbitMQURL)

	if err := mgr.Start(ctx); err != nil {
		slog.Error("manager exited", "error", err)
		os.Exit(1)
	}
}

func newK8sClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

type appConfig struct {
	RabbitMQURL      string
	Snapshot         workspace.SnapshotConfig
	Ingress          workspace.IngressConfig
	ResticImage      string
	IdleTimeout      int
	SnapshotSchedule string
}

func loadConfig() appConfig {
	return appConfig{
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672"),
		ResticImage: getEnv("RESTIC_IMAGE", "restic/restic:latest"),
		Snapshot: workspace.SnapshotConfig{
			Enabled:          getEnv("SNAPSHOT_ENABLED", "true") == "true",
			Repository:       getEnv("RESTIC_REPOSITORY", "s3:s3.amazonaws.com/nimbuscore-snapshots"),
			Password:         getEnv("RESTIC_PASSWORD", ""),
			S3Endpoint:       getEnv("AWS_ENDPOINT", ""),
			S3Bucket:         getEnv("AWS_BUCKET", "nimbuscore-snapshots"),
			S3Region:         getEnv("AWS_REGION", "us-east-1"),
			S3AccessKey:      getEnv("AWS_ACCESS_KEY_ID", ""),
			S3SecretKey:      getEnv("AWS_SECRET_ACCESS_KEY", ""),
			ScheduleInterval: getEnv("SNAPSHOT_INTERVAL", "1h"),
			RetentionDays:    30,
		},
		Ingress: workspace.IngressConfig{
			Enabled:   getEnv("INGRESS_ENABLED", "true") == "true",
			Host:      getEnv("INGRESS_HOST", "dev.nimbuscore.io"),
			TLSSecret: getEnv("INGRESS_TLS_SECRET", ""),
		},
		IdleTimeout:     20,
		SnapshotSchedule: getEnv("SNAPSHOT_SCHEDULE", "*/30 * * * *"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
