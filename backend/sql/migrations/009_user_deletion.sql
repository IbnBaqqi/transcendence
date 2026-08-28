-- +goose Up
ALTER TABLE users ADD COLUMN deleted_at timestamptz;

CREATE INDEX idx_users_active ON users (id) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_users_active;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;