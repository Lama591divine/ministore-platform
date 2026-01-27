CREATE TABLE IF NOT EXISTS products (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
    );

INSERT INTO products (id, title, price_cents) VALUES
    ('p1', 'Keyboard', 4990),
    ('p2', 'Mouse', 1990)
    ON CONFLICT (id) DO NOTHING;
