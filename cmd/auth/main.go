package main

import (
	"MiniStore/internal/auth"
	"MiniStore/pkg/kit"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const serviceName = "auth"

type Config struct {
	Port         string
	JWTSecret    string
	PostgresDSN  string
	MetricsToken string
}

func main() {
	log := kit.NewLogger(serviceName)
	defer func() {
		_ = log.Sync()
	}()

	if err := run(log); err != nil {
		log.Fatal("service failed", zap.Error(err))
	}
}

func run(log *zap.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	db, err := openPostgres(cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	if err := pingDB(db, 3*time.Second); err != nil {
		return err
	}

	h := buildHTTPHandler(log, cfg, db)

	return kit.RunHTTPServer(":"+cfg.Port, h, log)
}

func loadConfig() (Config, error) {
	cfg := Config{
		Port:         getenv("PORT", "8081"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		PostgresDSN:  os.Getenv("POSTGRES_DSN"),
		MetricsToken: os.Getenv("METRICS_TOKEN"),
	}

	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET is required and must be at least 32 chars")
	}
	if cfg.PostgresDSN == "" {
		return Config{}, errors.New("POSTGRES_DSN is required")
	}
	return cfg, nil
}

func openPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	return db, nil
}

func pingDB(db *sql.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return db.PingContext(ctx)
}

func buildHTTPHandler(log *zap.Logger, cfg Config, db *sql.DB) http.Handler {
	svc := &auth.Server{
		Log:   log,
		Store: auth.NewPostgresStore(db),
		JWT:   auth.NewTokenMaker(cfg.JWTSecret),
	}

	reg := prometheus.NewRegistry()

	return auth.NewHandler(svc, auth.HTTPDeps{
		Log:            log,
		Service:        serviceName,
		Registry:       reg,
		MetricsEnabled: true,
		MetricsToken:   cfg.MetricsToken,
	})
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
