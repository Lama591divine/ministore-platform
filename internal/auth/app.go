package auth

import (
	"net/http"
	"time"

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

const (
	loginLimitPerMin    = 5
	registerLimitPerMin = 3
	limitWindow         = 60 * time.Second
)

func NewHandler(s *Server, deps HTTPDeps) http.Handler {
	r := chi.NewRouter()

	metricsOn := deps.MetricsEnabled && deps.Registry != nil
	if deps.MetricsEnabled && deps.Registry == nil && deps.Log != nil {
		deps.Log.Warn("metrics enabled but Registry is nil")
	}

	setupMiddleware(r, deps, metricsOn)
	setupRoutes(r, s, deps, metricsOn)

	return r
}

func setupMiddleware(r *chi.Mux, deps HTTPDeps, metricsOn bool) {
	r.Use(chimw.RequestID)
	r.Use(kit.Recoverer)
	r.Use(kit.Logging(deps.Log))

	if metricsOn {
		metrics := kit.NewMetrics(deps.Registry)
		r.Use(metrics.Middleware(deps.Service, kit.ChiRoutePatternOrPath))
	}
}

func setupRoutes(r *chi.Mux, s *Server, deps HTTPDeps, metricsOn bool) {
	loginLimiter := kit.NewIPRateLimiter(loginLimitPerMin, int(limitWindow.Seconds()))
	registerLimiter := kit.NewIPRateLimiter(registerLimitPerMin, int(limitWindow.Seconds()))

	r.Route("/auth", func(rr chi.Router) {
		rr.With(loginLimiter.Middleware).Post("/login", s.handleLogin)
		rr.With(registerLimiter.Middleware).Post("/register", s.handleRegister)
		rr.Get("/whoami", s.handleWhoAmI)
	})

	r.Get("/healthz", healthz)
	r.Get("/readyz", s.handleReady)

	if metricsOn {
		r.With(kit.MetricsAuth(deps.MetricsToken)).Handle(
			"/metrics",
			promhttp.HandlerFor(deps.Registry, promhttp.HandlerOpts{}),
		)
	}
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
