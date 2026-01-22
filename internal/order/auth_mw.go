package order

import (
	"context"
	"net/http"
	"strings"

	"MiniStore/pkg/kit"
)

type ctxKey string

const userKey ctxKey = "user"

type User struct {
	ID   string
	Role string
}

func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey).(User)
	return u, ok
}

func RequireUserHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := strings.TrimSpace(r.Header.Get("X-User-Id"))
		if uid == "" {
			kit.WriteError(w, r, http.StatusUnauthorized, "missing X-User-Id", nil)
			return
		}
		role := strings.TrimSpace(r.Header.Get("X-User-Role"))
		ctx := context.WithValue(r.Context(), userKey, User{ID: uid, Role: role})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
