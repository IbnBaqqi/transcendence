-- +goose Up
UPDATE users
SET username = btrim(username, E' \t\n\r\v\f\u00a0'),
    email = btrim(email, E' \t\n\r\v\f\u00a0')
WHERE username <> btrim(username, E' \t\n\r\v\f\u00a0')
    OR email <> btrim(email, E' \t\n\r\v\f\u00a0');

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_uq;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_uq;

CREATE UNIQUE INDEX users_username_lower_uq ON users (lower(username));
CREATE UNIQUE INDEX users_email_lower_uq ON users (lower(email));

-- +goose Down
DROP INDEX IF EXISTS users_username_lower_uq;
DROP INDEX IF EXISTS users_email_lower_uq;

ALTER TABLE users ADD CONSTRAINT users_username_uq UNIQUE (username);
ALTER TABLE users ADD CONSTRAINT users_email_uq UNIQUE (email);
