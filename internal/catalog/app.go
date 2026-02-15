package catalog

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"MiniStore/pkg/kit"
)

type HTTPDeps struct {
	Log      *zap.Logger
	Service  string
	Registry *prometheus.Registry

	MetricsEnabled bool
	MetricsToken   string
}

func NewHandler(s *Server, deps HTTPDeps) http.Handler {
	r := chi.NewRouter()

	setupMiddleware(r, deps)
	setupMetrics(r, deps)

	r.Mount("/", s.Routes())
	return r
}

func setupMiddleware(r *chi.Mux, deps HTTPDeps) {
	r.Use(chimw.RequestID)
	r.Use(kit.Recoverer)
	r.Use(kit.Logging(deps.Log))
}

func setupMetrics(r *chi.Mux, deps HTTPDeps) {
	if deps.Registry == nil {
		return
	}

	metrics := kit.NewMetrics(deps.Registry)
	r.Use(metrics.Middleware(deps.Service, kit.ChiRoutePatternOrPath))

	if !deps.MetricsEnabled {
		return
	}

	r.With(kit.MetricsAuth(deps.MetricsToken)).
		Handle("/metrics", promhttp.HandlerFor(deps.Registry, promhttp.HandlerOpts{}))
}
