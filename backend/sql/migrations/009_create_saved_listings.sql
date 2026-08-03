-- +goose Up
CREATE TABLE IF NOT EXISTS saved_listings (
    -- Note the two different id types: user.id is uuid, listings.id is serial int.
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    listing_id integer NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,

    -- The pair IS the key: the DB now refuses a duplicate save outright.
    PRIMARY KEY (user_id, listing_id)
);

-- The PK already indexes (user_id, ...) so "my wishlist" is fast. This second
-- index covers the other direction - e.g. "how many people saved this listing".
CREATE INDEX idx_saved_listings_listing_id ON saved_listings(listing_id);

-- +goose Down
DROP TABLE IF EXISTS saved_listings;