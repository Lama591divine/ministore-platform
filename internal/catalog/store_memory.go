package catalog

import (
	"context"
	"sort"
	"sync"
)

type MemStore struct {
	mu       sync.RWMutex
	products map[string]Product
}

func NewMemStore() *MemStore {
	return &MemStore{
		products: map[string]Product{
			"p1": {ID: "p1", Title: "Keyboard", PriceCents: 4990},
			"p2": {ID: "p2", Title: "Mouse", PriceCents: 1990},
		},
	}
}

func (s *MemStore) Ping(ctx context.Context) error {
	return nil
}

func (s *MemStore) ListSortedByID(ctx context.Context) ([]Product, error) {
	s.mu.RLock()
	out := make([]Product, 0, len(s.products))
	for _, p := range s.products {
		out = append(out, p)
	}
	s.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *MemStore) Get(ctx context.Context, id string) (Product, bool, error) {
	s.mu.RLock()
	p, ok := s.products[id]
	s.mu.RUnlock()

	return p, ok, nil
}
