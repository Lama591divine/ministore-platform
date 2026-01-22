package main

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"MiniStore/internal/auth"
	"MiniStore/internal/gateway"
	"MiniStore/pkg/kit"
)

func main() {
	service := "gateway"
	log := kit.NewLogger(service)
	defer log.Sync()

	port := getenv("PORT", "8080")
	jwtSecret := getenv("JWT_SECRET", "dev-secret")

	authURL := getenv("AUTH_URL", "http://localhost:8081")
	catalogURL := getenv("CATALOG_URL", "http://localhost:8082")
	orderURL := getenv("ORDER_URL", "http://localhost:8083")

	metrics := kit.NewMetrics()

	authProxy, err := gateway.ReverseProxy(authURL)
	if err != nil {
		panic(err)
	}
	catalogProxy, err := gateway.ReverseProxy(catalogURL)
	if err != nil {
		panic(err)
	}
	orderProxy, err := gateway.ReverseProxy(orderURL)
	if err != nil {
		panic(err)
	}

	j := auth.NewTokenMaker(jwtSecret)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(kit.Recoverer)
	r.Use(kit.Logging(log))
	r.Use(metrics.Middleware(service, func(r *http.Request) string { return r.URL.Path }))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	r.Handle("/auth", authProxy)
	r.Handle("/auth/*", authProxy)

	r.Handle("/products", catalogProxy)
	r.Handle("/products/*", catalogProxy)

	r.Group(func(pr chi.Router) {
		pr.Use(gateway.AuthJWT(j))
		pr.Use(gateway.InjectHeaders)

		pr.Handle("/orders", orderProxy)
		pr.Handle("/orders/*", orderProxy)
	})

	_ = kit.RunHTTPServer(":"+port, r, log)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
