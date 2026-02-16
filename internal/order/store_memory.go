package order

import (
	"context"
	"sync"
)

type MemStore struct {
	mu     sync.RWMutex
	orders map[string]Order
}

func NewMemStore() *MemStore {
	return &MemStore{
		orders: make(map[string]Order),
	}
}

func (s *MemStore) Ping(ctx context.Context) error {
	return nil
}

func (s *MemStore) Create(ctx context.Context, o Order) error {
	s.mu.Lock()
	s.orders[o.ID] = o
	s.mu.Unlock()
	return nil
}

func (s *MemStore) Get(ctx context.Context, id string) (Order, bool, error) {
	s.mu.RLock()
	o, ok := s.orders[id]
	s.mu.RUnlock()

	if !ok {
		return Order{}, false, nil
	}

	return o, true, nil
}
