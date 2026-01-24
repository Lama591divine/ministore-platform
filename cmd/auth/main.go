package main

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"MiniStore/internal/auth"
	"MiniStore/pkg/kit"
)

func main() {
	service := "auth"
	log := kit.NewLogger(service)
	defer log.Sync()

	port := getenv("PORT", "8081")
	jwtSecret := getenv("JWT_SECRET", "dev-secret")

	metrics := kit.NewMetrics()

	s := &auth.Server{
		Log:   log,
		Store: auth.NewStore(),
		JWT:   auth.NewTokenMaker(jwtSecret),
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
	r.Mount("/", s.Routes())

	_ = kit.RunHTTPServer(":"+port, r, log)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
