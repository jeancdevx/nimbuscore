package cache

import (
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedis(url, password string, db int) *redis.Client {
	opts, err := redis.ParseURL(url)
	if err != nil {
		opts = &redis.Options{
			Addr:         url,
			Password:     password,
			DB:           db,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}
	}
	return redis.NewClient(opts)
}
