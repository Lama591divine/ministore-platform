package kit

import (
	"net/http"
	"strings"
)

func MetricsAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			authz := r.Header.Get("Authorization")
			if !strings.HasPrefix(authz, "Bearer ") {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			if strings.TrimPrefix(authz, "Bearer ") != token {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
