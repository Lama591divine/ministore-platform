package main

import (
	"context"
	"database/sql"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"MiniStore/internal/auth"
	"MiniStore/pkg/kit"
)

func main() {
	service := "auth"
	log := kit.NewLogger(service)
	defer func() { _ = log.Sync() }()

	port := getenv("PORT", "8081")
	jwtSecret := getenv("JWT_SECRET", "dev-secret")

	dsn := getenv("POSTGRES_DSN", "")
	if dsn == "" {
		log.Fatal("POSTGRES_DSN is required")
	}

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

	s := &auth.Server{
		Log:   log,
		Store: auth.NewPostgresStore(db),
		JWT:   auth.NewTokenMaker(jwtSecret),
	}

	reg := prometheus.NewRegistry()
	h := auth.NewHandler(s, auth.HTTPDeps{Log: log, Service: service, Registry: reg})

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
