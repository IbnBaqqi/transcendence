-- +goose Up
ALTER TABLE profiles ADD COLUMN location VARCHAR(100);

-- +goose Down
ALTER TABLE profiles DROP COLUMN location;