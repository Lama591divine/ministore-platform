package order

import (
	"context"
	"database/sql"
	"time"
)

const (
	pingTimeout   = 1 * time.Second
	createTimeout = 5 * time.Second
	getTimeout    = 5 * time.Second
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

func (s *PostgresStore) Create(ctx context.Context, o Order) error {
	return withTimeout(ctx, createTimeout, func(ctx context.Context) error {
		tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if err != nil {
			return err
		}

		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()

		if err := insertOrder(ctx, tx, o); err != nil {
			return err
		}

		if err := insertOrderItems(ctx, tx, o.ID, o.Items); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		committed = true
		return nil
	})
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Order, bool, error) {
	var o Order

	err := withTimeout(ctx, getTimeout, func(ctx context.Context) error {
		if err := s.db.QueryRowContext(ctx, `
			SELECT id, user_id, total_cents, status, created_at
			FROM orders
			WHERE id = $1
		`, id).Scan(&o.ID, &o.UserID, &o.TotalCents, &o.Status, &o.CreatedAt); err != nil {
			return err
		}

		items, err := loadOrderItems(ctx, s.db, id)
		if err != nil {
			return err
		}

		o.Items = items
		return nil
	})

	if err == sql.ErrNoRows {
		return Order{}, false, nil
	}
	if err != nil {
		return Order{}, false, err
	}

	return o, true, nil
}

func insertOrder(ctx context.Context, tx *sql.Tx, o Order) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orders (id, user_id, total_cents, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, o.ID, o.UserID, o.TotalCents, o.Status, o.CreatedAt)
	return err
}

func insertOrderItems(ctx context.Context, tx *sql.Tx, orderID string, items []Item) error {
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO order_items (order_id, product_id, qty)
		VALUES ($1, $2, $3)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, it := range items {
		if _, err := stmt.ExecContext(ctx, orderID, it.ProductID, it.Qty); err != nil {
			return err
		}
	}

	return nil
}

func loadOrderItems(ctx context.Context, q queryer, orderID string) ([]Item, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT product_id, qty
		FROM order_items
		WHERE order_id = $1
		ORDER BY product_id ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Item, 0, 8)
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ProductID, &it.Qty); err != nil {
			return nil, err
		}
		out = append(out, it)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func withTimeout(parent context.Context, d time.Duration, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(parent, d)
	defer cancel()
	return fn(ctx)
}
