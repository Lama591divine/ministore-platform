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

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET is required and must be at least 32 chars")
	}

	deps := gateway.Deps{
		JWTSecret:  jwtSecret,
		AuthURL:    getenv("AUTH_URL", "http://auth:8081"),
		CatalogURL: getenv("CATALOG_URL", "http://catalog:8082"),
		OrderURL:   getenv("ORDER_URL", "http://order:8083"),
	}

	reg := prometheus.NewRegistry()
	h, err := gateway.NewHandler(deps, gateway.HTTPDeps{
		Log:            log,
		Service:        service,
		Registry:       reg,
		MetricsEnabled: true,
		MetricsToken:   os.Getenv("METRICS_TOKEN"),
	})
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
