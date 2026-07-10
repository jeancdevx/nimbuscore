module github.com/nimbuscore/pkg/auth

go 1.26.5

require (
	github.com/coreos/go-oidc/v3 v3.13.0
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/uuid v1.6.0
	golang.org/x/oauth2 v0.29.0
)

require (
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	golang.org/x/crypto v0.36.0 // indirect
)

replace github.com/nimbuscore/pkg/api => ../api
