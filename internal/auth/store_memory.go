package auth

import (
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type MemStore struct {
	mu      sync.RWMutex
	byEmail map[string]User
}

func NewMemStore() *MemStore {
	return &MemStore{byEmail: make(map[string]User)}
}

func (s *MemStore) Create(email, password, role, id string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byEmail[email]; ok {
		return ErrEmailExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	s.byEmail[email] = User{ID: id, Email: email, Hash: hash, Role: role}
	return nil
}

func (s *MemStore) Verify(email, password string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

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
