-- +goose Up
ALTER TABLE profiles ALTER COLUMN firstname DROP NOT NULL;
ALTER TABLE profiles ALTER COLUMN lastname DROP NOT NULL;

INSERT INTO profiles (id)
SELECT id FROM users
ON CONFLICT (id) DO NOTHING;

-- +goose Down
UPDATE profiles SET firstname = '' WHERE firstname IS NULL;
UPDATE profiles SET lastname = '' WHERE lastname IS NULL;

ALTER TABLE profiles ALTER COLUMN firstname SET NOT NULL;
ALTER TABLE profiles ALTER COLUMN lastname SET NOT NULL;
