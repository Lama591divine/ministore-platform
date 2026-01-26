package catalog

type Product struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	PriceCents int64  `json:"price_cents"`
}

type Store interface {
	ListSortedByID() ([]Product, error)
	Get(id string) (Product, bool, error)
}
