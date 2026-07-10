package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimbuscore/pkg/api"
)

func ListAuditLogs(ctx context.Context, db *pgxpool.Pool, limit, offset int) ([]api.AuditEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := db.Query(ctx, `
		SELECT id, action, user_id, target_id, target_type, metadata, ip_address, created_at
		FROM audit_log ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []api.AuditEntry
	for rows.Next() {
		var e api.AuditEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.UserID, &e.TargetID, &e.TargetType, &e.Metadata, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func ListAuditLogsByUser(ctx context.Context, db *pgxpool.Pool, userID uuid.UUID, limit, offset int) ([]api.AuditEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := db.Query(ctx, `
		SELECT id, action, user_id, target_id, target_type, metadata, ip_address, created_at
		FROM audit_log WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []api.AuditEntry
	for rows.Next() {
		var e api.AuditEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.UserID, &e.TargetID, &e.TargetType, &e.Metadata, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func CreateBillingRecord(ctx context.Context, db *pgxpool.Pool, rec *api.BillingRecord) error {
	_, err := db.Exec(ctx, `
		INSERT INTO billing_records (id, workspace_id, team_id, user_id, cpu_cores, memory_gb,
			disk_gb, gpu_count, duration_min, cost, started_at, ended_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, rec.ID, rec.WorkspaceID, rec.TeamID, rec.UserID, rec.CPUCores, rec.MemoryGB,
		rec.DiskGB, rec.GPUCount, rec.DurationMin, rec.Cost, rec.StartedAt, rec.EndedAt, time.Now().UTC())
	return err
}

func GetBillingByTeam(ctx context.Context, db *pgxpool.Pool, teamID uuid.UUID, since, until time.Time) ([]api.BillingRecord, error) {
	rows, err := db.Query(ctx, `
		SELECT id, workspace_id, team_id, user_id, cpu_cores, memory_gb, disk_gb,
			gpu_count, duration_min, cost, started_at, ended_at, created_at
		FROM billing_records WHERE team_id = $1 AND started_at >= $2 AND started_at <= $3
		ORDER BY started_at DESC
	`, teamID, since, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []api.BillingRecord
	for rows.Next() {
		var r api.BillingRecord
		if err := rows.Scan(&r.ID, &r.WorkspaceID, &r.TeamID, &r.UserID, &r.CPUCores, &r.MemoryGB,
			&r.DiskGB, &r.GPUCount, &r.DurationMin, &r.Cost, &r.StartedAt, &r.EndedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

func ListClusters(ctx context.Context, db *pgxpool.Pool) ([]api.Cluster, error) {
	rows, err := db.Query(ctx, `SELECT id, name, api_endpoint, region, provider, enabled, labels, created_at, updated_at FROM clusters ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clusters []api.Cluster
	for rows.Next() {
		var c api.Cluster
		if err := rows.Scan(&c.ID, &c.Name, &c.APIEndpoint, &c.Region, &c.Provider, &c.Enabled, &c.Labels, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		clusters = append(clusters, c)
	}
	return clusters, nil
}

func GetCluster(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*api.Cluster, error) {
	var c api.Cluster
	err := db.QueryRow(ctx, `SELECT id, name, api_endpoint, region, provider, enabled, labels, created_at, updated_at FROM clusters WHERE id = $1`, id).
		Scan(&c.ID, &c.Name, &c.APIEndpoint, &c.Region, &c.Provider, &c.Enabled, &c.Labels, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, api.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func CreateCluster(ctx context.Context, db *pgxpool.Pool, c *api.Cluster) error {
	_, err := db.Exec(ctx, `
		INSERT INTO clusters (id, name, api_endpoint, region, provider, enabled, labels, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, c.ID, c.Name, c.APIEndpoint, c.Region, c.Provider, c.Enabled, c.Labels, time.Now().UTC(), time.Now().UTC())
	return err
}

func DeleteCluster(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) error {
	_, err := db.Exec(ctx, `DELETE FROM clusters WHERE id = $1`, id)
	return err
}
