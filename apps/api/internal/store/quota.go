package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimbuscore/pkg/api"
)

func GetQuota(ctx context.Context, db *pgxpool.Pool, teamID uuid.UUID) (*api.Quota, error) {
	var q api.Quota
	err := db.QueryRow(ctx, `
		SELECT team_id, max_workspaces, max_cpu, max_memory, max_disk, max_gpu, updated_at
		FROM quotas WHERE team_id = $1
	`, teamID).Scan(&q.TeamID, &q.MaxWorkspaces, &q.MaxCPU, &q.MaxMemory, &q.MaxDisk, &q.MaxGPU, &q.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, api.ErrNotFound
		}
		return nil, err
	}

	err = db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(cpu)::TEXT, '0'), COALESCE(SUM(memory)::TEXT, '0'), COALESCE(SUM(disk)::TEXT, '0')
		FROM workspaces WHERE team_id = $1 AND status NOT IN ('deleted', 'stopped')
	`, teamID).Scan(&q.UsedWorkspaces, &q.UsedCPU, &q.UsedMemory, &q.UsedDisk)
	if err != nil {
		q.UsedWorkspaces = 0
	}

	return &q, nil
}

func UpsertQuota(ctx context.Context, db *pgxpool.Pool, q *api.Quota) error {
	_, err := db.Exec(ctx, `
		INSERT INTO quotas (team_id, max_workspaces, max_cpu, max_memory, max_disk, max_gpu, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (team_id) DO UPDATE SET
			max_workspaces = EXCLUDED.max_workspaces,
			max_cpu = EXCLUDED.max_cpu,
			max_memory = EXCLUDED.max_memory,
			max_disk = EXCLUDED.max_disk,
			max_gpu = EXCLUDED.max_gpu,
			updated_at = EXCLUDED.updated_at
	`, q.TeamID, q.MaxWorkspaces, q.MaxCPU, q.MaxMemory, q.MaxDisk, q.MaxGPU)
	return err
}

func DeleteQuota(ctx context.Context, db *pgxpool.Pool, teamID uuid.UUID) error {
	_, err := db.Exec(ctx, `DELETE FROM quotas WHERE team_id = $1`, teamID)
	return err
}
