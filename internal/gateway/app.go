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
	Registry *prometheus.Registry // nil => без метрик
}

type Deps struct {
	AuthURL    string
	CatalogURL string
	OrderURL   string
	JWTSecret  string
}

func NewHandler(deps Deps, httpDeps HTTPDeps) (http.Handler, error) {
	authProxy, err := NewReverseProxy(deps.AuthURL, httpDeps.Log)
	if err != nil {
		return nil, err
	}
	catalogProxy, err := NewReverseProxy(deps.CatalogURL, httpDeps.Log)
	if err != nil {
		return nil, err
	}
	orderProxy, err := NewReverseProxy(deps.OrderURL, httpDeps.Log)
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

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 900*time.Millisecond)
		defer cancel()

		if err := checkReady(ctx, deps.AuthURL+"/readyz"); err != nil {
			if httpDeps.Log != nil {
				httpDeps.Log.Warn("readyz failed: auth", zap.Error(err))
			}
			kit.WriteError(w, r, http.StatusServiceUnavailable, "auth not ready", nil)
			return
		}
		if err := checkReady(ctx, deps.CatalogURL+"/readyz"); err != nil {
			if httpDeps.Log != nil {
				httpDeps.Log.Warn("readyz failed: catalog", zap.Error(err))
			}
			kit.WriteError(w, r, http.StatusServiceUnavailable, "catalog not ready", nil)
			return
		}
		if err := checkReady(ctx, deps.OrderURL+"/readyz"); err != nil {
			if httpDeps.Log != nil {
				httpDeps.Log.Warn("readyz failed: order", zap.Error(err))
			}
			kit.WriteError(w, r, http.StatusServiceUnavailable, "order not ready", nil)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

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

func checkReady(ctx context.Context, url string) error {
	client := &http.Client{
		Timeout: 600 * time.Millisecond,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
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
