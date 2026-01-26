package catalog

import (
	"sort"
	"sync"
)

type MemStore struct {
	mu sync.RWMutex
	m  map[string]Product
}

func NewMemStore() *MemStore {
	s := &MemStore{m: map[string]Product{}}
	s.m["p1"] = Product{ID: "p1", Title: "Keyboard", PriceCents: 4990}
	s.m["p2"] = Product{ID: "p2", Title: "Mouse", PriceCents: 1990}
	return s
}

func NewStore() Store {
	return NewMemStore()
}

func (s *MemStore) ListSortedByID() ([]Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Product, 0, len(s.m))
	for _, p := range s.m {
		out = append(out, p)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *MemStore) Get(id string) (Product, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.m[id]
	return p, ok, nil
}
