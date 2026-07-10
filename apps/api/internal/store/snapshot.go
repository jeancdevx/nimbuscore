package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimbuscore/pkg/api"
)

func ListSnapshots(ctx context.Context, db *pgxpool.Pool, workspaceID uuid.UUID) ([]api.Snapshot, error) {
	rows, err := db.Query(ctx, `
		SELECT id, workspace_id, status, size, message, started_at, completed_at, created_at
		FROM snapshots WHERE workspace_id = $1 ORDER BY created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snaps []api.Snapshot
	for rows.Next() {
		var s api.Snapshot
		if err := rows.Scan(&s.ID, &s.WorkspaceID, &s.Status, &s.Size, &s.Message, &s.StartedAt, &s.CompletedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		snaps = append(snaps, s)
	}
	return snaps, nil
}

func GetSnapshot(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*api.Snapshot, error) {
	var s api.Snapshot
	err := db.QueryRow(ctx, `
		SELECT id, workspace_id, status, size, message, started_at, completed_at, created_at
		FROM snapshots WHERE id = $1
	`, id).Scan(&s.ID, &s.WorkspaceID, &s.Status, &s.Size, &s.Message, &s.StartedAt, &s.CompletedAt, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, api.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func CreateSnapshot(ctx context.Context, db *pgxpool.Pool, s *api.Snapshot) error {
	_, err := db.Exec(ctx, `
		INSERT INTO snapshots (id, workspace_id, status, size, message, started_at, completed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, s.ID, s.WorkspaceID, s.Status, s.Size, s.Message, s.StartedAt, s.CompletedAt, s.CreatedAt)
	return err
}

func UpdateSnapshotStatus(ctx context.Context, db *pgxpool.Pool, id uuid.UUID, status api.SnapshotStatus, size int64, message string) error {
	_, err := db.Exec(ctx, `
		UPDATE snapshots SET status = $1, size = $2, message = $3, completed_at = NOW()
		WHERE id = $4
	`, status, size, message, id)
	return err
}
