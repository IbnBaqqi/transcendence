-- +goose Up
CREATE TABLE IF NOT EXISTS listing_images (
    id serial PRIMARY KEY,
    listing_id integer NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    filename text NOT NULL UNIQUE,
    position integer NOT NULL DEFAULT 0,
    CREATED_AT timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_listing_images_listing_id ON listing_images(listing_id);

-- +goose Down
DROP TABLE IF EXISTS listing_images;