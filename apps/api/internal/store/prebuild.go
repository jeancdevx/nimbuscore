package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimbuscore/pkg/api"
)

func ListPrebuilds(ctx context.Context, db *pgxpool.Pool) ([]api.Prebuild, error) {
	rows, err := db.Query(ctx, `
		SELECT id, user_id, team_id, name, image, repo_url, branch, devcontainer_path,
			   status, message, created_at, updated_at
		FROM prebuilds ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prebuilds []api.Prebuild
	for rows.Next() {
		var p api.Prebuild
		if err := rows.Scan(&p.ID, &p.UserID, &p.TeamID, &p.Name, &p.Image, &p.RepoURL, &p.Branch,
			&p.DevcontainerPath, &p.Status, &p.Message, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prebuilds = append(prebuilds, p)
	}
	return prebuilds, nil
}

func GetPrebuild(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*api.Prebuild, error) {
	var p api.Prebuild
	err := db.QueryRow(ctx, `
		SELECT id, user_id, team_id, name, image, repo_url, branch, devcontainer_path,
			   status, message, created_at, updated_at
		FROM prebuilds WHERE id = $1
	`, id).Scan(&p.ID, &p.UserID, &p.TeamID, &p.Name, &p.Image, &p.RepoURL, &p.Branch,
		&p.DevcontainerPath, &p.Status, &p.Message, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, api.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func CreatePrebuild(ctx context.Context, db *pgxpool.Pool, p *api.Prebuild) error {
	_, err := db.Exec(ctx, `
		INSERT INTO prebuilds (id, user_id, team_id, name, image, repo_url, branch,
			devcontainer_path, status, message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, p.ID, p.UserID, p.TeamID, p.Name, p.Image, p.RepoURL, p.Branch,
		p.DevcontainerPath, p.Status, p.Message, p.CreatedAt, p.UpdatedAt)
	return err
}

func UpdatePrebuildStatus(ctx context.Context, db *pgxpool.Pool, id uuid.UUID, status api.PrebuildStatus, message string) error {
	_, err := db.Exec(ctx, `UPDATE prebuilds SET status = $1, message = $2, updated_at = NOW() WHERE id = $3`, status, message, id)
	return err
}

func DeletePrebuild(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) error {
	_, err := db.Exec(ctx, `DELETE FROM prebuilds WHERE id = $1`, id)
	return err
}
