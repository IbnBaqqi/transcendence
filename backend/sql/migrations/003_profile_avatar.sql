-- +goose Up
ALTER TABLE profiles ADD COLUMN avatar_filename text;

-- Mirrors listing_images_filename_key. Postgres allows many NULLs in a unique
ALTER TABLE profiles
    ADD CONSTRAINT profiles_avatar_filename_key UNIQUE (avatar_filename);

-- +goose Down
ALTER TABLE profiles DROP COLUMN avatar_filename;
