-- +goose Up
UPDATE users SET role = 'USER' WHERE role NOT IN ('USER', 'ADMIN');

ALTER TABLE users
    ADD CONSTRAINT users_role_check CHECK (role IN ('USER', 'ADMIN'));

-- +goose Down
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
