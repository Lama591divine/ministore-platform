package main

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"MiniStore/internal/gateway"
	"MiniStore/pkg/kit"
)

func main() {
	service := "gateway"
	log := kit.NewLogger(service)
	defer func() { _ = log.Sync() }()

	port := getenv("PORT", "8080")

	deps := gateway.Deps{
		JWTSecret:  getenv("JWT_SECRET", "dev-secret"),
		AuthURL:    getenv("AUTH_URL", "http://localhost:8081"),
		CatalogURL: getenv("CATALOG_URL", "http://localhost:8082"),
		OrderURL:   getenv("ORDER_URL", "http://localhost:8083"),
	}

	reg := prometheus.NewRegistry()
	h, err := gateway.NewHandler(deps, gateway.HTTPDeps{Log: log, Service: service, Registry: reg})
	if err != nil {
		log.Fatal("init gateway handler failed", zap.Error(err))
	}

	if err := kit.RunHTTPServer(":"+port, h, log); err != nil {
		log.Fatal("http server stopped", zap.Error(err))
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
