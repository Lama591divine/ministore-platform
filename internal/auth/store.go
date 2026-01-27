package auth

import (
	"context"
	"errors"
)

var (
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type User struct {
	ID    string
	Email string
	Hash  []byte
	Role  string
}

type UserStore interface {
	Create(ctx context.Context, email, password, role, id string) error
	Verify(ctx context.Context, email, password string) (User, error)
	Ping(ctx context.Context) error
}
