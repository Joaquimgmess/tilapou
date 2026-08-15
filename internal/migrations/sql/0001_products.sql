-- +goose Up
CREATE TABLE products (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    price_cents BIGINT NOT NULL CHECK (price_cents > 0),
    created_at  TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE products;
