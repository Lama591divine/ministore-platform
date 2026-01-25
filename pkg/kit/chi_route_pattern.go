package kit

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func ChiRoutePatternOrPath(r *http.Request) string {
	if rp := chi.RouteContext(r.Context()).RoutePattern(); rp != "" {
		return rp
	}
	return r.URL.Path
}
