-- +goose Up

ALTER TABLE users ADD COLUMN is_visible boolean NOT NULL
    GENERATED ALWAYS AS (deleted_at IS NULL AND suspended_at IS NULL) STORED;

CREATE INDEX users_visible_idx ON users (id) WHERE is_visible;

-- +goose Down

DROP INDEX IF EXISTS users_visible_idx;

ALTER TABLE users DROP COLUMN is_visible;
