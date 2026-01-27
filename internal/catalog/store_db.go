package catalog

import (
	"context"
	"database/sql"
	"time"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	return s.db.PingContext(ctx)
}

func (s *PostgresStore) ListSortedByID(ctx context.Context) ([]Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, price_cents
		FROM products
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Product, 0, 16)
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Title, &p.PriceCents); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Product, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var p Product
	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, price_cents
		FROM products
		WHERE id = $1
	`, id).Scan(&p.ID, &p.Title, &p.PriceCents)

	if err == sql.ErrNoRows {
		return Product{}, false, nil
	}
	if err != nil {
		return Product{}, false, err
	}
	return p, true, nil
}
