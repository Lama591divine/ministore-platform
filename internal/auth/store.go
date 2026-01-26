package auth

import "errors"

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
	Create(email, password, role, id string) error
	Verify(email, password string) (User, error)
}
