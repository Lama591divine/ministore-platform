package main

import (
	"database/sql"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
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
	dsn := getenv("POSTGRES_DSN", "")

	var store catalog.Store
	if dsn != "" {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			log.Fatal("db open", zap.Error(err))
		}
		if err := db.Ping(); err != nil {
			log.Fatal("db ping", zap.Error(err))
		}
		store = catalog.NewPostgresStore(db)
	} else {
		store = catalog.NewMemStore()
	}

	s := &catalog.Server{Store: store}

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
