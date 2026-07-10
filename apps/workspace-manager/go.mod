module github.com/nimbuscore/apps/workspace-manager

go 1.26.5

require (
	github.com/google/uuid v1.6.0
	github.com/nimbuscore/pkg/api v0.0.0
	github.com/nimbuscore/pkg/workspace v0.0.0
	github.com/rabbitmq/amqp091-go v1.10.0
	k8s.io/api v0.32.3
	k8s.io/apimachinery v0.32.3
	k8s.io/client-go v0.32.3
)

replace (
	github.com/nimbuscore/pkg/api => ../../pkg/api
	github.com/nimbuscore/pkg/workspace => ../../pkg/workspace
)
