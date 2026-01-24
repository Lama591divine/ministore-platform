package main

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"MiniStore/internal/order"
	"MiniStore/pkg/kit"
)

func main() {
	service := "order"
	log := kit.NewLogger(service)
	defer func() { _ = log.Sync() }()

	port := getenv("PORT", "8083")
	catalogURL := getenv("CATALOG_URL", "http://localhost:8082")

	metrics := kit.NewMetrics()

	s := &order.Server{
		Store:   order.NewStore(),
		Catalog: order.NewCatalogClient(catalogURL),
	}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(kit.Recoverer)
	r.Use(kit.Logging(log))

	r.Use(metrics.Middleware(service, func(r *http.Request) string {
		if rp := chi.RouteContext(r.Context()).RoutePattern(); rp != "" {
			return rp
		}
		return r.URL.Path
	}))

	r.Handle("/metrics", promhttp.Handler())

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	r.Group(func(pr chi.Router) {
		pr.Use(order.RequireUserHeaders)
		pr.Post("/orders", s.CreateHandler())
		pr.Get("/orders/{id}", s.GetHandler())
	})

	if err := kit.RunHTTPServer(":"+port, r, log); err != nil {
		log.Fatal("http server stopped", zap.Error(err))
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
