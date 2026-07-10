package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimbuscore/pkg/api"
	"github.com/nimbuscore/pkg/auth"
)

type auditContextKey string

const auditDBKey auditContextKey = "audit:db"

func WithAuditDB(ctx context.Context, db *pgxpool.Pool) context.Context {
	return context.WithValue(ctx, auditDBKey, db)
}

func GetAuditDB(ctx context.Context) *pgxpool.Pool {
	db, _ := ctx.Value(auditDBKey).(*pgxpool.Pool)
	return db
}

func AuditLogger(db *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := WithAuditDB(r.Context(), db)
			ctx = context.WithValue(ctx, ipKey, getClientIP(r))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type ipCtxKey string

const ipKey ipCtxKey = "audit:ip"

func GetClientIP(ctx context.Context) string {
	ip, _ := ctx.Value(ipKey).(string)
	return ip
}

func getClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	if real := r.Header.Get("X-Real-IP"); real != "" {
		return real
	}
	return r.RemoteAddr
}

func LogAuditEntry(ctx context.Context, action api.AuditAction, targetID, targetType, metadata string) {
	db := GetAuditDB(ctx)
	if db == nil {
		return
	}

	claims := auth.GetClaims(ctx)
	var userID uuid.UUID
	if claims != nil {
		userID = claims.UserID
	}

	ipAddr := GetClientIP(ctx)

	var tgtID, tgtType, meta *string
	if targetID != "" {
		tgtID = &targetID
	}
	if targetType != "" {
		tgtType = &targetType
	}
	if metadata != "" {
		meta = &metadata
	}

	_, err := db.Exec(ctx, `
		INSERT INTO audit_log (id, action, user_id, target_id, target_type, metadata, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, uuid.New(), string(action), userID, tgtID, tgtType, meta, ipAddr, time.Now().UTC())
	if err != nil {
		slog.Warn("failed to write audit log", "action", action, "error", err)
	}
}
