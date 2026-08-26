-- +goose Up
ALTER TABLE profiles ADD COLUMN avatar_filename text;

-- Mirrors listing_images_filename_key. NULLs are distinct in a unique index, so
-- this constrains real filenames without stopping every user who has no avatar
-- from having none at the same time.
ALTER TABLE profiles
    ADD CONSTRAINT profiles_avatar_filename_key UNIQUE (avatar_filename);

-- +goose Down
ALTER TABLE profiles DROP COLUMN avatar_filename;
