package main

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"MiniStore/internal/catalog"
	"MiniStore/pkg/kit"
)

func main() {
	service := "catalog"
	log := kit.NewLogger(service)
	defer log.Sync()

	port := getenv("PORT", "8082")

	metrics := kit.NewMetrics()
	s := &catalog.Server{Store: catalog.NewStore()}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(kit.Recoverer)
	r.Use(kit.Logging(log))
	r.Use(metrics.Middleware(service, func(r *http.Request) string { return r.URL.Path }))

	r.Handle("/metrics", promhttp.Handler())
	r.Mount("/", s.Routes())

	_ = kit.RunHTTPServer(":"+port, r, log)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
