package main

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
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

	s := &order.Server{
		Store:   order.NewStore(),
		Catalog: order.NewCatalogClient(catalogURL),
	}

	reg := prometheus.NewRegistry()
	h := order.NewHandler(s, order.HTTPDeps{Log: log, Service: service, Registry: reg})

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
