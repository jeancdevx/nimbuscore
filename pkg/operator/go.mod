module github.com/nimbuscore/pkg/operator

go 1.26.5

require (
	github.com/nimbuscore/pkg/api v0.0.0
	k8s.io/api v0.32.3
	k8s.io/apimachinery v0.32.3
	sigs.k8s.io/controller-runtime v0.20.4
)

replace github.com/nimbuscore/pkg/api => ../api
