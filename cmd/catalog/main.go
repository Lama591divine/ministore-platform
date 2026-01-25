package main

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"MiniStore/internal/catalog"
	"MiniStore/pkg/kit"
)

func main() {
	service := "catalog"
	log := kit.NewLogger(service)
	defer func() { _ = log.Sync() }()

	port := getenv("PORT", "8082")

	s := &catalog.Server{Store: catalog.NewStore()}

	reg := prometheus.NewRegistry()
	h := catalog.NewHandler(s, catalog.HTTPDeps{Log: log, Service: service, Registry: reg})

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
