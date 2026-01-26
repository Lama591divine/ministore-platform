package order

import "time"

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
	Create(o Order) error
	Get(id string) (Order, bool, error)
}
