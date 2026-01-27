package order

import (
	"context"
	"time"
)

type Item struct {
	ProductID string `json:"product_id"`
	Qty       int    `json:"qty"`
}

type Order struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Items      []Item    `json:"items"`
	TotalCents int64     `json:"total_cents"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type Store interface {
	Create(ctx context.Context, o Order) error
	Get(ctx context.Context, id string) (Order, bool, error)
	Ping(ctx context.Context) error
}
