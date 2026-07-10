package snapshot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/nimbuscore/pkg/workspace"
)

type Service struct {
	cfg         workspace.SnapshotConfig
	resticImage string
}

func New(cfg workspace.SnapshotConfig, resticImage string) *Service {
	return &Service{cfg: cfg, resticImage: resticImage}
}

func (s *Service) Backup(ctx context.Context, workspaceID, podName, namespace string) error {
	if !s.cfg.Enabled {
		slog.Info("snapshots disabled, skipping backup", "workspace_id", workspaceID)
		return nil
	}

	slog.Info("starting snapshot backup", "workspace_id", workspaceID)

	args := []string{
		"exec", "-n", namespace, podName, "--",
		"restic", "-r", s.cfg.Repository,
		"--host", workspaceID,
		"backup", "/home/coder",
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = append(os.Environ(), s.resticEnv()...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restic backup failed: %w\noutput: %s", err, string(output))
	}

	slog.Info("snapshot backup completed", "workspace_id", workspaceID, "output", string(output))
	return nil
}

func (s *Service) Restore(ctx context.Context, workspaceID, podName, namespace string) error {
	if !s.cfg.Enabled {
		return nil
	}

	slog.Info("starting snapshot restore", "workspace_id", workspaceID)

	args := []string{
		"exec", "-n", namespace, podName, "--",
		"restic", "-r", s.cfg.Repository,
		"--host", workspaceID,
		"restore", "latest", "--target", "/",
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = append(os.Environ(), s.resticEnv()...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restic restore failed: %w\noutput: %s", err, string(output))
	}

	slog.Info("snapshot restore completed", "workspace_id", workspaceID, "output", string(output))
	return nil
}

func (s *Service) resticEnv() []string {
	env := []string{
		"RESTIC_PASSWORD=" + s.cfg.Password,
		"AWS_ACCESS_KEY_ID=" + s.cfg.S3AccessKey,
		"AWS_SECRET_ACCESS_KEY=" + s.cfg.S3SecretKey,
	}
	if s.cfg.S3Endpoint != "" {
		env = append(env, "RESTIC_REPOSITORY_S3_ENDPOINT="+s.cfg.S3Endpoint)
	}
	if s.cfg.S3Region != "" {
		env = append(env, "AWS_DEFAULT_REGION="+s.cfg.S3Region)
	}
	return env
}
