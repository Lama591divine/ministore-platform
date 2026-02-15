package auth

import (
	"context"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type MemStore struct {
	mu      sync.RWMutex
	byEmail map[string]User
}

func NewMemStore() *MemStore {
	return &MemStore{
		byEmail: make(map[string]User),
	}
}

func (s *MemStore) Ping(context.Context) error { return nil }

func (s *MemStore) Create(_ context.Context, email, password, role, id string) error {
	email = normalizeEmail(email)
	password = normalizePassword(password)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byEmail[email]; exists {
		return ErrEmailExists
	}

	s.byEmail[email] = User{
		ID:    id,
		Email: email,
		Hash:  hash,
		Role:  role,
	}
	return nil
}

func (s *MemStore) Verify(_ context.Context, email, password string) (User, error) {
	email = normalizeEmail(email)
	password = normalizePassword(password)

	s.mu.RLock()
	u, ok := s.byEmail[email]
	s.mu.RUnlock()

	if !ok {
		return User{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword(u.Hash, []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}
	return u, nil
}
