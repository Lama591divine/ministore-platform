package gateway

import (
	"net/http"

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
	Registry *prometheus.Registry // nil => без метрик
}

type Deps struct {
	AuthURL    string
	CatalogURL string
	OrderURL   string
	JWTSecret  string
}

func NewHandler(deps Deps, httpDeps HTTPDeps) (http.Handler, error) {
	authProxy, err := ReverseProxy(deps.AuthURL)
	if err != nil {
		return nil, err
	}
	catalogProxy, err := ReverseProxy(deps.CatalogURL)
	if err != nil {
		return nil, err
	}
	orderProxy, err := ReverseProxy(deps.OrderURL)
	if err != nil {
		return nil, err
	}

	j := auth.NewTokenMaker(deps.JWTSecret)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(kit.Recoverer)
	r.Use(kit.Logging(httpDeps.Log))

	if httpDeps.Registry != nil {
		metrics := kit.NewMetrics(httpDeps.Registry)
		r.Use(metrics.Middleware(httpDeps.Service, kit.ChiRoutePatternOrPath))
		r.Handle("/metrics", promhttp.HandlerFor(httpDeps.Registry, promhttp.HandlerOpts{}))
	}

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	r.Handle("/auth", authProxy)
	r.Handle("/auth/*", authProxy)

	r.Handle("/products", catalogProxy)
	r.Handle("/products/*", catalogProxy)

	r.Group(func(pr chi.Router) {
		pr.Use(AuthJWT(j))
		pr.Use(InjectHeaders)

		pr.Handle("/orders", orderProxy)
		pr.Handle("/orders/*", orderProxy)
	})

	return r, nil
}
