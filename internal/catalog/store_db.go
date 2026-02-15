package catalog

import (
	"context"
	"database/sql"
	"time"
)

const (
	pingTimeout  = 1 * time.Second
	queryTimeout = 3 * time.Second
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

func (s *PostgresStore) ListSortedByID(ctx context.Context) ([]Product, error) {
	var out []Product

	err := withTimeout(ctx, queryTimeout, func(ctx context.Context) error {
		rows, err := s.db.QueryContext(ctx, `
			SELECT id, title, price_cents
			FROM products
			ORDER BY id ASC
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		out = make([]Product, 0, 16)
		for rows.Next() {
			var p Product
			if err := rows.Scan(&p.ID, &p.Title, &p.PriceCents); err != nil {
				return err
			}
			out = append(out, p)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Product, bool, error) {
	var (
		p   Product
		err error
	)

	err = withTimeout(ctx, queryTimeout, func(ctx context.Context) error {
		return s.db.QueryRowContext(ctx, `
			SELECT id, title, price_cents
			FROM products
			WHERE id = $1
		`, id).Scan(&p.ID, &p.Title, &p.PriceCents)
	})

	if err == sql.ErrNoRows {
		return Product{}, false, nil
	}
	if err != nil {
		return Product{}, false, err
	}
	return p, true, nil
}

func withTimeout(parent context.Context, d time.Duration, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(parent, d)
	defer cancel()
	return fn(ctx)
}
