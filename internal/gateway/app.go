package gateway

import (
	"context"
	"fmt"
	"io"
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

	MetricsEnabled bool
	MetricsToken   string
}

type Deps struct {
	AuthURL    string
	CatalogURL string
	OrderURL   string
	JWTSecret  string
}

const (
	readyTimeout      = 2 * time.Second
	readyProbeTimeout = 700 * time.Millisecond
)

var readyClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	},
}

func NewHandler(deps Deps, httpDeps HTTPDeps) (http.Handler, error) {
	authProxy, catalogProxy, orderProxy, err := buildProxies(deps, httpDeps.Log)
	if err != nil {
		return nil, err
	}

	jwt := auth.NewTokenMaker(deps.JWTSecret)

	r := chi.NewRouter()
	setupMiddleware(r, httpDeps)
	setupMetrics(r, httpDeps)

	r.Get("/healthz", healthz)
	r.Get("/readyz", readyz(deps, httpDeps.Log))

	r.Handle("/auth", authProxy)
	r.Handle("/auth/*", authProxy)

	r.Handle("/products", catalogProxy)
	r.Handle("/products/*", catalogProxy)

	r.Group(func(pr chi.Router) {
		pr.Use(AuthJWT(jwt))
		pr.Handle("/orders", orderProxy)
		pr.Handle("/orders/*", orderProxy)
	})

	return r, nil
}

func buildProxies(deps Deps, log *zap.Logger) (authProxy, catalogProxy, orderProxy http.Handler, err error) {
	ap, err := NewReverseProxy(deps.AuthURL, log)
	if err != nil {
		return nil, nil, nil, err
	}

	cp, err := NewReverseProxy(deps.CatalogURL, log)
	if err != nil {
		return nil, nil, nil, err
	}

	op, err := NewReverseProxy(deps.OrderURL, log)
	if err != nil {
		return nil, nil, nil, err
	}

	return ap, cp, op, nil
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

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func readyz(deps Deps, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()

		if err := checkReady(ctx, deps.AuthURL+"/readyz"); err != nil {
			if log != nil {
				log.Warn("readyz failed: auth", zap.Error(err))
			}
			kit.WriteError(w, r, http.StatusServiceUnavailable, "auth not ready", nil)
			return
		}

		if err := checkReady(ctx, deps.CatalogURL+"/readyz"); err != nil {
			if log != nil {
				log.Warn("readyz failed: catalog", zap.Error(err))
			}
			kit.WriteError(w, r, http.StatusServiceUnavailable, "catalog not ready", nil)
			return
		}

		if err := checkReady(ctx, deps.OrderURL+"/readyz"); err != nil {
			if log != nil {
				log.Warn("readyz failed: order", zap.Error(err))
			}
			kit.WriteError(w, r, http.StatusServiceUnavailable, "order not ready", nil)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func checkReady(ctx context.Context, url string) error {
	cctx, cancel := context.WithTimeout(ctx, readyProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := readyClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status=%d", resp.StatusCode)
	}

	return nil
}
