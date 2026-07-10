package middleware

import (
	"context"
	"net/http"

	"github.com/nimbuscore/pkg/api"
	"github.com/nimbuscore/pkg/auth"
)

type contextKey string

const permissionsKey contextKey = "rbac:permissions"

func WithPermissions(ctx context.Context, perms []api.Permission) context.Context {
	return context.WithValue(ctx, permissionsKey, perms)
}

func GetPermissions(ctx context.Context) []api.Permission {
	perms, _ := ctx.Value(permissionsKey).([]api.Permission)
	return perms
}

func HasPermission(ctx context.Context, perm api.Permission) bool {
	perms := GetPermissions(ctx)
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

func RequirePermission(perm api.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.GetClaims(r.Context())
			if claims == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			rolePerms, ok := api.RolePermissions[api.Role(claims.Role)]
			if !ok {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			ctx := WithPermissions(r.Context(), rolePerms)

			hasPerm := false
			for _, p := range rolePerms {
				if p == perm {
					hasPerm = true
					break
				}
			}

			if !hasPerm {
				http.Error(w, "insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RBACInjector(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := auth.GetClaims(r.Context())
		if claims != nil {
			rolePerms, ok := api.RolePermissions[api.Role(claims.Role)]
			if ok {
				ctx := WithPermissions(r.Context(), rolePerms)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
