-- +goose Up

-- One row = one order: a buyer purchasing some quantity of a listing, tracked
-- through its lifecycle. Mirrors the listings table style (SERIAL id, NUMERIC
-- money, text+CHECK enum).
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    listing_id INTEGER NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    buyer_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    seller_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(10, 2) NOT NULL CHECK (unit_price > 0),
    total_price NUMERIC(10,2) NOT NULL CHECK (total_price > 0),
    status TEXT NOT NULL DEFAULT 'pending'
            CHECK (status IN ('pending', 'confirmed', 'paid', 'completed', 'cancelled')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- we filter orders by buyer or seller (the "my orders" list) and look them up
-- by listing, so index those foreign keys.
CREATE INDEX idx_orders_buyer_id ON orders(buyer_id);
CREATE INDEX idx_orders_seller_id ON orders(seller_id);
CREATE INDEX idx_orders_listing_id ON orders(listing_id);

-- +goose Down
DROP TABLE IF EXISTS orders;