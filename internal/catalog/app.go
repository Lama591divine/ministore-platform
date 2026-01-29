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
	r.Use(chimw.RequestID)
	r.Use(kit.Recoverer)
	r.Use(kit.Logging(deps.Log))

	if deps.Registry != nil {
		metrics := kit.NewMetrics(deps.Registry)
		r.Use(metrics.Middleware(deps.Service, kit.ChiRoutePatternOrPath))

		if deps.MetricsEnabled {
			r.With(kit.MetricsAuth(deps.MetricsToken)).
				Handle("/metrics", promhttp.HandlerFor(deps.Registry, promhttp.HandlerOpts{}))
		}
	}

	r.Mount("/", s.Routes())
	return r
}
