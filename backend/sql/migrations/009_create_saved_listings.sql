-- +goose Up
CREATE TABLE IF NOT EXISTS saved_listings (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    listing_id integer NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (user_id, listing_id)
);

CREATE INDEX idx_saved_listings_listing_id ON saved_listings(listing_id);

-- +goose Down
DROP TABLE IF EXISTS saved_listings;
