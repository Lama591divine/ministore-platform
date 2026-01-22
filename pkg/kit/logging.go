package kit

import "go.uber.org/zap"

func NewLogger(service string) *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.InitialFields = map[string]any{"service": service}
	l, _ := cfg.Build()
	return l
}
