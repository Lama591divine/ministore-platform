package gateway

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"MiniStore/internal/auth"
	"MiniStore/pkg/kit"
)

type ctxKey string

const (
	userIDKey   ctxKey = "user_id"
	userRoleKey ctxKey = "user_role"
)

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok
}

func UserRoleFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userRoleKey).(string)
	return v, ok
}

func AuthJWT(jwt *auth.TokenMaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if !strings.HasPrefix(authz, "Bearer ") {
				kit.WriteError(w, r, http.StatusUnauthorized, "missing token", nil)
				return
			}
			claims, err := jwt.Parse(strings.TrimPrefix(authz, "Bearer "))
			if err != nil {
				kit.WriteError(w, r, http.StatusUnauthorized, "invalid token", nil)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			ctx = context.WithValue(ctx, userRoleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ReverseProxy(target string) (*httputil.ReverseProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	return httputil.NewSingleHostReverseProxy(u), nil
}

func InjectHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-User-Id")
		r.Header.Del("X-User-Role")

		if uid, ok := UserIDFromContext(r.Context()); ok && uid != "" {
			r.Header.Set("X-User-Id", uid)
		}
		if role, ok := UserRoleFromContext(r.Context()); ok && role != "" {
			r.Header.Set("X-User-Role", role)
		}

		next.ServeHTTP(w, r)
	})
}
