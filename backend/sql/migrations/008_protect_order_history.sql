-- +goose Up

ALTER TABLE orders DROP CONSTRAINT orders_listing_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_listing_id_fkey
    FOREIGN KEY (listing_id) REFERENCES listings(id) ON DELETE RESTRICT;

-- +goose Down

ALTER TABLE orders DROP CONSTRAINT orders_listing_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_listing_id_fkey
    FOREIGN KEY (listing_id) REFERENCES listings(id) ON DELETE CASCADE;
