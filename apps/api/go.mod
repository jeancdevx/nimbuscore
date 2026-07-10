module github.com/nimbuscore/apps/api

go 1.26.5

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/go-chi/cors v1.2.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.4
	github.com/nimbuscore/pkg/api v0.0.0
	github.com/nimbuscore/pkg/auth v0.0.0
	github.com/nimbuscore/pkg/workspace v0.0.0
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/redis/go-redis/v9 v9.7.3
)

replace (
	github.com/nimbuscore/pkg/api => ../../pkg/api
	github.com/nimbuscore/pkg/auth => ../../pkg/auth
	github.com/nimbuscore/pkg/workspace => ../../pkg/workspace
)
