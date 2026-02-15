package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

const (
	pingTimeout  = 1 * time.Second
	queryTimeout = 3 * time.Second
	pgUniqueCode = "23505"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	return withTimeout(ctx, pingTimeout, func(ctx context.Context) error {
		return s.db.PingContext(ctx)
	})
}

func (s *PostgresStore) Create(ctx context.Context, email, password, role, id string) error {
	email = normalizeEmail(email)
	password = normalizePassword(password)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return withTimeout(ctx, queryTimeout, func(ctx context.Context) error {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO users (id, email, pass_hash, role)
			VALUES ($1, $2, $3, $4)
		`, id, email, hash, role)

		if err == nil {
			return nil
		}
		if isUniqueViolation(err) {
			return ErrEmailExists
		}
		return err
	})
}

func (s *PostgresStore) Verify(ctx context.Context, email, password string) (User, error) {
	email = normalizeEmail(email)
	password = normalizePassword(password)

	var u User
	err := withTimeout(ctx, queryTimeout, func(ctx context.Context) error {
		return s.db.QueryRowContext(ctx, `
			SELECT id, email, pass_hash, role
			FROM users
			WHERE email = $1
		`, email).Scan(&u.ID, &u.Email, &u.Hash, &u.Role)
	})
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

func withTimeout(parent context.Context, d time.Duration, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(parent, d)
	defer cancel()
	return fn(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueCode
}
