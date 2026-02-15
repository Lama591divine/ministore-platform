package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"MiniStore/internal/catalog"
	"MiniStore/pkg/kit"
)

const serviceName = "catalog"

type Config struct {
	Port          string
	PostgresDSN   string
	AllowMemStore bool

	MetricsEnabled bool
	MetricsToken   string
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

	store, cleanup, err := buildStore(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	srv := &catalog.Server{
		Store: store,
		Log:   log,
	}

	reg := prometheus.NewRegistry()
	h := catalog.NewHandler(srv, catalog.HTTPDeps{
		Log:            log,
		Service:        serviceName,
		Registry:       reg,
		MetricsEnabled: cfg.MetricsEnabled,
		MetricsToken:   cfg.MetricsToken,
	})

	return kit.RunHTTPServer(":"+cfg.Port, h, log)
}

func loadConfig() (Config, error) {
	cfg := Config{
		Port:          getenv("PORT", "8082"),
		PostgresDSN:   os.Getenv("POSTGRES_DSN"),
		AllowMemStore: os.Getenv("ALLOW_MEMSTORE") == "1",

		MetricsEnabled: true,
		MetricsToken:   os.Getenv("METRICS_TOKEN"),
	}

	if cfg.PostgresDSN == "" && !cfg.AllowMemStore {
		return Config{}, errors.New("POSTGRES_DSN is required (set ALLOW_MEMSTORE=1 for dev)")
	}

	return cfg, nil
}

func buildStore(cfg Config) (catalog.Store, func(), error) {
	if cfg.PostgresDSN == "" {
		return catalog.NewMemStore(), func() {}, nil
	}

	db, err := openPostgres(cfg.PostgresDSN)
	if err != nil {
		return nil, func() {}, err
	}

	if err := pingDB(db, 3*time.Second); err != nil {
		_ = db.Close()
		return nil, func() {}, err
	}

	return catalog.NewPostgresStore(db), func() { _ = db.Close() }, nil
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

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
