-- +goose Up

CREATE INDEX orders_created_at_idx ON orders (created_at DESC, id DESC);

-- +goose Down

DROP INDEX IF EXISTS orders_created_at_idx;
