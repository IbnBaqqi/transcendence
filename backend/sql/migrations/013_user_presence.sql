-- +goose Up
ALTER TABLE users ADD COLUMN last_seen_at timestamp;
ALTER TABLE users ADD COLUMN show_online_status boolean NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE users DROP COLUMN show_online_status;
ALTER TABLE users DROP COLUMN last_seen_at;