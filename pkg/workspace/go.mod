module github.com/nimbuscore/pkg/workspace

go 1.26.5

require (
	github.com/google/uuid v1.6.0
	github.com/nimbuscore/pkg/api v0.0.0
	github.com/rabbitmq/amqp091-go v1.10.0
)

replace github.com/nimbuscore/pkg/api => ../api
