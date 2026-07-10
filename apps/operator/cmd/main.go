package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/nimbuscore/apps/operator/internal/controller"
	nimbuscorev1alpha1 "github.com/nimbuscore/pkg/operator"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(nimbuscorev1alpha1.AddToScheme(scheme))
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: ":8080",
		},
		HealthProbeBindAddress: ":8081",
		LeaderElection:         true,
		LeaderElectionID:       "nimbuscore-operator-leader",
	})
	if err != nil {
		slog.Error("failed to create manager", "error", err)
		os.Exit(1)
	}

	ingressHost := getEnv("INGRESS_HOST", "dev.nimbuscore.io")
	tlsSecret := getEnv("INGRESS_TLS_SECRET", "")
	outputRegistry := getEnv("OUTPUT_REGISTRY", "ghcr.io/nimbuscore")

	if err := controller.RegisterWorkspaceController(mgr, ingressHost, tlsSecret); err != nil {
		slog.Error("failed to register workspace controller", "error", err)
		os.Exit(1)
	}

	if err := controller.RegisterPrebuildController(mgr, outputRegistry); err != nil {
		slog.Error("failed to register prebuild controller", "error", err)
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		slog.Error("failed to add health check", "error", err)
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		slog.Error("failed to add ready check", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	slog.Info("starting operator")
	if err := mgr.Start(ctx); err != nil {
		slog.Error("manager exited", "error", err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
