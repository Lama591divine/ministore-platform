package auth

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

	loginLimiter := kit.NewIPRateLimiter(5, 1*60)
	registerLimiter := kit.NewIPRateLimiter(3, 1*60)

	r.Group(func(rr chi.Router) {
		rr.With(loginLimiter.Middleware).Post("/auth/login", s.handleLogin)
		rr.With(registerLimiter.Middleware).Post("/auth/register", s.handleRegister)
	})

	r.Get("/auth/whoami", s.handleWhoAmI)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", s.handleReady)

	if deps.MetricsEnabled && deps.Registry != nil {
		metrics := kit.NewMetrics(deps.Registry)
		r.Use(metrics.Middleware(deps.Service, kit.ChiRoutePatternOrPath))

		r.With(kit.MetricsAuth(deps.MetricsToken)).Handle(
			"/metrics",
			promhttp.HandlerFor(deps.Registry, promhttp.HandlerOpts{}),
		)
	}

	return r
}
