package auth

import (
	"errors"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID    string
	Email string
	Hash  []byte
	Role  string
}

type Store struct {
	mu      sync.RWMutex
	byEmail map[string]User
}

func NewStore() *Store {
	return &Store{byEmail: make(map[string]User)}
}

func (s *Store) Create(email, password, role, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byEmail[email]; ok {
		return errors.New("email already exists")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.byEmail[email] = User{ID: id, Email: email, Hash: hash, Role: role}
	return nil
}

func (s *Store) Verify(email, password string) (User, error) {
	s.mu.RLock()
	u, ok := s.byEmail[email]
	s.mu.RUnlock()
	if !ok {
		return User{}, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword(u.Hash, []byte(password)); err != nil {
		return User{}, errors.New("invalid credentials")
	}
	return u, nil
}
