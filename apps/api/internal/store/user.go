package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimbuscore/pkg/api"
)

func ListUsers(ctx context.Context, db *pgxpool.Pool) ([]api.User, error) {
	rows, err := db.Query(ctx, `SELECT id, email, name, provider, provider_id, avatar_url, created_at, updated_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []api.User
	for rows.Next() {
		var u api.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Provider, &u.ProviderID, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func GetUser(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*api.User, error) {
	var u api.User
	err := db.QueryRow(ctx, `SELECT id, email, name, provider, provider_id, avatar_url, created_at, updated_at FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Email, &u.Name, &u.Provider, &u.ProviderID, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, api.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func UpsertUser(ctx context.Context, db *pgxpool.Pool, u *api.User) error {
	_, err := db.Exec(ctx, `
		INSERT INTO users (id, email, name, provider, provider_id, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (provider, provider_id) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			avatar_url = EXCLUDED.avatar_url,
			updated_at = EXCLUDED.updated_at
	`, u.ID, u.Email, u.Name, u.Provider, u.ProviderID, u.AvatarURL, u.CreatedAt, u.UpdatedAt)
	return err
}
