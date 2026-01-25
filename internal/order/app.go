package order

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
}

func NewHandler(s *Server, deps HTTPDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(kit.Recoverer)
	r.Use(kit.Logging(deps.Log))

	if deps.Registry != nil {
		metrics := kit.NewMetrics(deps.Registry)
		r.Use(metrics.Middleware(deps.Service, kit.ChiRoutePatternOrPath))
		r.Handle("/metrics", promhttp.HandlerFor(deps.Registry, promhttp.HandlerOpts{}))
	}

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	r.Group(func(pr chi.Router) {
		pr.Use(RequireUserHeaders)
		pr.Post("/orders", s.CreateHandler())
		pr.Get("/orders/{id}", s.GetHandler())
	})

	return r
}
