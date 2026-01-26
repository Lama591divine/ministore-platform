package order

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

func (s *PostgresStore) Create(o Order) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (id, user_id, total_cents, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, o.ID, o.UserID, o.TotalCents, o.Status, o.CreatedAt)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO order_items (order_id, product_id, qty)
		VALUES ($1, $2, $3)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, it := range o.Items {
		if _, err := stmt.ExecContext(ctx, o.ID, it.ProductID, it.Qty); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) Get(id string) (Order, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var o Order
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, total_cents, status, created_at
		FROM orders
		WHERE id = $1
	`, id).Scan(&o.ID, &o.UserID, &o.TotalCents, &o.Status, &o.CreatedAt)

	if err == sql.ErrNoRows {
		return Order{}, false, nil
	}
	if err != nil {
		return Order{}, false, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT product_id, qty
		FROM order_items
		WHERE order_id = $1
		ORDER BY product_id ASC
	`, id)
	if err != nil {
		return Order{}, false, err
	}
	defer rows.Close()

	items := make([]Item, 0, 8)
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ProductID, &it.Qty); err != nil {
			return Order{}, false, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return Order{}, false, err
	}
	o.Items = items

	return o, true, nil
}
