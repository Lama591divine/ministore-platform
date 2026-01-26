package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Create(email, password, role, id string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, pass_hash, role)
		VALUES ($1, $2, $3, $4)
	`, id, email, hash, role)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailExists
		}
		return err
	}

	return nil
}

func (s *PostgresStore) Verify(email, password string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var u User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, pass_hash, role
		FROM users
		WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.Hash, &u.Role)

	if err == sql.ErrNoRows {
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, err
	}

	if err := bcrypt.CompareHashAndPassword(u.Hash, []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}

	return u, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
