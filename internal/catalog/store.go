package catalog

import "sync"

type Product struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	PriceCents int64  `json:"price_cents"`
}

type Store struct {
	mu sync.RWMutex
	m  map[string]Product
}

func NewStore() *Store {
	s := &Store{m: map[string]Product{}}
	s.m["p1"] = Product{ID: "p1", Title: "Keyboard", PriceCents: 4990}
	s.m["p2"] = Product{ID: "p2", Title: "Mouse", PriceCents: 1990}
	return s
}

func (s *Store) List() []Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Product, 0, len(s.m))
	for _, p := range s.m {
		out = append(out, p)
	}
	return out
}

func (s *Store) Get(id string) (Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.m[id]
	return p, ok
}
