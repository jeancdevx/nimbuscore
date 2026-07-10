package server

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nimbuscore/apps/api/internal/queue"
	"github.com/nimbuscore/pkg/api"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	cfg api.APIConfig
	db  *pgxpool.Pool
	rdb *redis.Client
	rmq *queue.RabbitMQ
}

func New(cfg api.APIConfig, db *pgxpool.Pool, rdb *redis.Client, rmq *queue.RabbitMQ) *Server {
	return &Server{
		cfg: cfg,
		db:  db,
		rdb: rdb,
		rmq: rmq,
	}
}
