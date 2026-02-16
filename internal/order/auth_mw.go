package order

import (
	"context"
	"net/http"
	"strings"

	"MiniStore/internal/auth"
	"MiniStore/pkg/kit"
)

type ctxKey string

const (
	userKey      ctxKey = "user"
	bearerPrefix        = "Bearer "
)

type User struct {
	ID   string
	Role string
}

func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey).(User)
	return u, ok
}

func AuthJWT(jwt *auth.TokenMaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok, ok := bearerToken(r)
			if !ok {
				kit.WriteError(w, r, http.StatusUnauthorized, "missing token", nil)
				return
			}

			claims, err := jwt.Parse(tok)
			if err != nil || claims.UserID == "" {
				kit.WriteError(w, r, http.StatusUnauthorized, "invalid token", nil)
				return
			}

			ctx := context.WithValue(r.Context(), userKey, User{ID: claims.UserID, Role: claims.Role})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(authz, bearerPrefix) {
		return "", false
	}

	tok := strings.TrimSpace(strings.TrimPrefix(authz, bearerPrefix))
	if tok == "" {
		return "", false
	}

	return tok, true
}
