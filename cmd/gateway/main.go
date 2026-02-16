package main

import (
	"errors"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"MiniStore/internal/gateway"
	"MiniStore/pkg/kit"
)

const serviceName = "gateway"

type Config struct {
	Port      string
	JWTSecret string

	AuthURL    string
	CatalogURL string
	OrderURL   string

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

	reg := prometheus.NewRegistry()
	h, err := gateway.NewHandler(
		gateway.Deps{
			JWTSecret:  cfg.JWTSecret,
			AuthURL:    cfg.AuthURL,
			CatalogURL: cfg.CatalogURL,
			OrderURL:   cfg.OrderURL,
		},
		gateway.HTTPDeps{
			Log:            log,
			Service:        serviceName,
			Registry:       reg,
			MetricsEnabled: cfg.MetricsEnabled,
			MetricsToken:   cfg.MetricsToken,
		},
	)
	if err != nil {
		return err
	}

	return kit.RunHTTPServer(":"+cfg.Port, h, log)
}

func loadConfig() (Config, error) {
	cfg := Config{
		Port:      getenv("PORT", "8080"),
		JWTSecret: os.Getenv("JWT_SECRET"),

		AuthURL:    getenv("AUTH_URL", "http://auth:8081"),
		CatalogURL: getenv("CATALOG_URL", "http://catalog:8082"),
		OrderURL:   getenv("ORDER_URL", "http://order:8083"),

		MetricsEnabled: true,
		MetricsToken:   os.Getenv("METRICS_TOKEN"),
	}

	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET is required and must be at least 32 chars")
	}

	return cfg, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
