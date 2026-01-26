package order

import "sync"

type MemStore struct {
	mu sync.RWMutex
	m  map[string]Order
}

func NewMemStore() *MemStore {
	return &MemStore{m: map[string]Order{}}
}

func NewStore() Store {
	return NewMemStore()
}

func (s *MemStore) Create(o Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[o.ID] = o
	return nil
}

func (s *MemStore) Get(id string) (Order, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.m[id]
	return o, ok, nil
}
