package order

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"MiniStore/internal/auth"
	"MiniStore/pkg/kit"
)

type HTTPDeps struct {
	Log      *zap.Logger
	Service  string
	Registry *prometheus.Registry

	JWTSecret string

	MetricsEnabled bool
	MetricsToken   string
}

func NewHandler(s *Server, deps HTTPDeps) http.Handler {
	if len(deps.JWTSecret) < 32 {
		panic("JWTSecret is required and must be at least 32 chars")
	}

	j := auth.NewTokenMaker(deps.JWTSecret)

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

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		if err := s.Store.Ping(ctx); err != nil {
			if s.Log != nil {
				s.Log.Warn("readyz failed", zap.Error(err))
			}
			kit.WriteError(w, r, http.StatusServiceUnavailable, "not ready", nil)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	r.Group(func(pr chi.Router) {
		pr.Use(AuthJWT(j))
		pr.Post("/orders", s.CreateHandler())
		pr.Get("/orders/{id}", s.GetHandler())
	})

	return r
}
