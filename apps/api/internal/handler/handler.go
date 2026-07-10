package handler

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimbuscore/apps/api/internal/queue"
	"github.com/nimbuscore/pkg/api"
	"github.com/nimbuscore/pkg/auth"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	cfg     api.APIConfig
	db      *pgxpool.Pool
	rdb     *redis.Client
	rmq     *queue.RabbitMQ
	jwtAuth *auth.JWTIssuer
}

func New(cfg api.APIConfig, db *pgxpool.Pool, rdb *redis.Client, rmq *queue.RabbitMQ, jwtAuth *auth.JWTIssuer) *Handler {
	return &Handler{cfg: cfg, db: db, rdb: rdb, rmq: rmq, jwtAuth: jwtAuth}
}
