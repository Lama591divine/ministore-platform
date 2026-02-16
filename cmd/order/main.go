package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"MiniStore/internal/order"
	"MiniStore/pkg/kit"
)

const serviceName = "order"

type Config struct {
	Port       string
	CatalogURL string
	JWTSecret  string

	PostgresDSN   string
	AllowMemStore bool

	MetricsEnabled bool
	MetricsToken   string
}

func main() {
	log := kit.NewLogger(serviceName)
	defer func() { _ = log.Sync() }()

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

	srv := &order.Server{
		Store:   store,
		Catalog: order.NewCatalogClient(cfg.CatalogURL),
		Log:     log,
	}

	reg := prometheus.NewRegistry()
	h := order.NewHandler(srv, order.HTTPDeps{
		Log:            log,
		Service:        serviceName,
		Registry:       reg,
		JWTSecret:      cfg.JWTSecret,
		MetricsEnabled: cfg.MetricsEnabled,
		MetricsToken:   cfg.MetricsToken,
	})

	return kit.RunHTTPServer(":"+cfg.Port, h, log)
}

func loadConfig() (Config, error) {
	cfg := Config{
		Port:       getenv("PORT", "8083"),
		CatalogURL: getenv("CATALOG_URL", "http://catalog:8082"),
		JWTSecret:  os.Getenv("JWT_SECRET"),

		PostgresDSN:   os.Getenv("POSTGRES_DSN"),
		AllowMemStore: os.Getenv("ALLOW_MEMSTORE") == "1",

		MetricsEnabled: true,
		MetricsToken:   os.Getenv("METRICS_TOKEN"),
	}

	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET is required and must be at least 32 chars")
	}

	if cfg.PostgresDSN == "" && !cfg.AllowMemStore {
		return Config{}, errors.New("POSTGRES_DSN is required (set ALLOW_MEMSTORE=1 for dev)")
	}

	if _, err := http.NewRequest(http.MethodGet, cfg.CatalogURL, nil); err != nil {
		return Config{}, errors.New("CATALOG_URL is invalid")
	}

	return cfg, nil
}

func buildStore(cfg Config) (order.Store, func(), error) {
	if cfg.PostgresDSN == "" {
		return order.NewMemStore(), func() {}, nil
	}

	db, err := openPostgres(cfg.PostgresDSN)
	if err != nil {
		return nil, func() {}, err
	}

	if err := pingDB(db, 3*time.Second); err != nil {
		_ = db.Close()
		return nil, func() {}, err
	}

	return order.NewPostgresStore(db), func() { _ = db.Close() }, nil
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
