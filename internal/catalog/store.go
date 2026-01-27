package catalog

import "context"

type Product struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	PriceCents int64  `json:"price_cents"`
}

type Store interface {
	ListSortedByID(ctx context.Context) ([]Product, error)
	Get(ctx context.Context, id string) (Product, bool, error)
	Ping(ctx context.Context) error
}
