package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimbuscore/pkg/api"
)

func ListWorkspacesByUser(ctx context.Context, db *pgxpool.Pool, userID uuid.UUID) ([]api.Workspace, error) {
	rows, err := db.Query(ctx, `
		SELECT id, name, user_id, team_id, image, status, cpu, memory, disk, gpu,
			   repo_url, branch, ports, last_activity, created_at, updated_at
		FROM workspaces WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []api.Workspace
	for rows.Next() {
		ws, err := scanWorkspace(rows)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, *ws)
	}
	return workspaces, nil
}

func GetWorkspace(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*api.Workspace, error) {
	rows, err := db.Query(ctx, `
		SELECT id, name, user_id, team_id, image, status, cpu, memory, disk, gpu,
			   repo_url, branch, ports, last_activity, created_at, updated_at
		FROM workspaces WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, api.ErrNotFound
	}

	ws, err := scanWorkspace(rows)
	if err != nil {
		return nil, err
	}
	return ws, nil
}

func CreateWorkspace(ctx context.Context, db *pgxpool.Pool, ws *api.Workspace) error {
	_, err := db.Exec(ctx, `
		INSERT INTO workspaces (id, name, user_id, team_id, image, status,
			cpu, memory, disk, gpu, repo_url, branch, ports, last_activity, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, ws.ID, ws.Name, ws.UserID, ws.TeamID, ws.Image, ws.Status,
		ws.Resources.CPU, ws.Resources.Memory, ws.Resources.Disk, ws.Resources.GPU,
		ws.RepoURL, ws.Branch, ws.Ports, ws.LastActivity, ws.CreatedAt, ws.UpdatedAt)
	return err
}

func UpdateWorkspaceStatus(ctx context.Context, db *pgxpool.Pool, id uuid.UUID, status api.WorkspaceStatus) error {
	_, err := db.Exec(ctx, `UPDATE workspaces SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	return err
}

func DeleteWorkspace(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) error {
	_, err := db.Exec(ctx, `UPDATE workspaces SET status = 'deleted', updated_at = NOW() WHERE id = $1`, id)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanWorkspace(row scanner) (*api.Workspace, error) {
	var ws api.Workspace
	err := row.Scan(
		&ws.ID, &ws.Name, &ws.UserID, &ws.TeamID, &ws.Image, &ws.Status,
		&ws.Resources.CPU, &ws.Resources.Memory, &ws.Resources.Disk, &ws.Resources.GPU,
		&ws.RepoURL, &ws.Branch, &ws.Ports, &ws.LastActivity, &ws.CreatedAt, &ws.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, api.ErrNotFound
		}
		return nil, err
	}
	return &ws, nil
}
