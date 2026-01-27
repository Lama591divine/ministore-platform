CREATE TABLE IF NOT EXISTS orders (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    total_cents BIGINT NOT NULL CHECK (total_cents >= 0),
    status      TEXT NOT NULL CHECK (status IN ('NEW','PAID','CANCELLED')),
    created_at  TIMESTAMPTZ NOT NULL
    );

CREATE TABLE IF NOT EXISTS order_items (
    order_id   TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL,
    qty        INTEGER NOT NULL CHECK (qty > 0),
    PRIMARY KEY (order_id, product_id)
    );

CREATE INDEX IF NOT EXISTS idx_orders_user_created_at
    ON orders(user_id, created_at DESC);
