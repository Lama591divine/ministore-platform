package main

import (
	"context"
	"database/sql"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
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
	catalogURL := getenv("CATALOG_URL", "http://catalog:8082")

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET is required and must be at least 32 chars")
	}

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" && os.Getenv("ALLOW_MEMSTORE") != "1" {
		log.Fatal("POSTGRES_DSN is required (set ALLOW_MEMSTORE=1 for dev)")
	}

	var store order.Store
	if dsn != "" {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			log.Fatal("db open", zap.Error(err))
		}
		defer func() { _ = db.Close() }()

		db.SetMaxOpenConns(20)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(30 * time.Minute)
		db.SetConnMaxIdleTime(5 * time.Minute)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			log.Fatal("db ping", zap.Error(err))
		}

		store = order.NewPostgresStore(db)
	} else {
		store = order.NewMemStore()
	}

	s := &order.Server{
		Store:   store,
		Catalog: order.NewCatalogClient(catalogURL),
		Log:     log,
	}

	reg := prometheus.NewRegistry()
	h := order.NewHandler(s, order.HTTPDeps{
		Log:            log,
		Service:        service,
		Registry:       reg,
		JWTSecret:      jwtSecret,
		MetricsEnabled: true,
		MetricsToken:   os.Getenv("METRICS_TOKEN"),
	})

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
