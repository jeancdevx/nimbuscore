package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimbuscore/pkg/api"
)

func ListTeams(ctx context.Context, db *pgxpool.Pool) ([]api.Team, error) {
	rows, err := db.Query(ctx, `SELECT id, name, slug, created_at, updated_at FROM teams ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teams := []api.Team{}
	for rows.Next() {
		var t api.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, nil
}

func GetTeam(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*api.Team, error) {
	var t api.Team
	err := db.QueryRow(ctx, `SELECT id, name, slug, created_at, updated_at FROM teams WHERE id = $1`, id).
		Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, api.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func CreateTeam(ctx context.Context, db *pgxpool.Pool, t *api.Team) error {
	_, err := db.Exec(ctx, `INSERT INTO teams (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		t.ID, t.Name, t.Slug, t.CreatedAt, t.UpdatedAt)
	return err
}

func UpdateTeam(ctx context.Context, db *pgxpool.Pool, id uuid.UUID, name, slug string) (*api.Team, error) {
	_, err := db.Exec(ctx, `UPDATE teams SET name = $1, slug = $2, updated_at = NOW() WHERE id = $3`, name, slug, id)
	if err != nil {
		return nil, err
	}
	return GetTeam(ctx, db, id)
}

func DeleteTeam(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) error {
	_, err := db.Exec(ctx, `DELETE FROM teams WHERE id = $1`, id)
	return err
}
