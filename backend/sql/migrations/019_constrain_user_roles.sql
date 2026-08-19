-- +goose Up
-- Fail closed, and deliberately lossy: a stray 'admin' becomes 'USER' and the
-- Down cannot restore it. Demoting an unreadable role is the safe direction,
-- and there are no admins yet - the README makes the first one by hand.
UPDATE users SET role = 'USER' WHERE role NOT IN ('USER', 'ADMIN');

ALTER TABLE users
    ADD CONSTRAINT users_role_check CHECK (role IN ('USER', 'ADMIN'));

-- +goose Down
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
