package order

import (
	"sync"
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

type Store struct {
	mu sync.RWMutex
	m  map[string]Order
}

func NewStore() *Store {
	return &Store{m: map[string]Order{}}
}

func (s *Store) Put(o Order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[o.ID] = o
}

func (s *Store) Get(id string) (Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.m[id]
	return o, ok
}
