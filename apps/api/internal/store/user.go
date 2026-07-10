package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimbuscore/pkg/api"
)

func ListUsers(ctx context.Context, db *pgxpool.Pool) ([]api.User, error) {
	rows, err := db.Query(ctx, `SELECT id, email, name, role, provider, provider_id, avatar_url, created_at, updated_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []api.User
	for rows.Next() {
		var u api.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Provider, &u.ProviderID, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func GetUser(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*api.User, error) {
	var u api.User
	err := db.QueryRow(ctx, `SELECT id, email, name, role, provider, provider_id, avatar_url, created_at, updated_at FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Provider, &u.ProviderID, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, api.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func UpsertUser(ctx context.Context, db *pgxpool.Pool, u *api.User) error {
	err := db.QueryRow(ctx, `
		INSERT INTO users (id, email, name, role, provider, provider_id, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (provider, provider_id) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			avatar_url = EXCLUDED.avatar_url,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`, u.ID, u.Email, u.Name, u.Role, u.Provider, u.ProviderID, u.AvatarURL, u.CreatedAt, u.UpdatedAt).Scan(&u.ID)
	return err
}

func UpdateUserRole(ctx context.Context, db *pgxpool.Pool, id uuid.UUID, role api.Role) error {
	_, err := db.Exec(ctx, `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`, string(role), id)
	return err
}
