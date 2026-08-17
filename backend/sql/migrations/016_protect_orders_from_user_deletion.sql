-- +goose Up
ALTER TABLE orders DROP CONSTRAINT orders_buyer_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_buyer_id_fkey
    FOREIGN KEY (buyer_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE orders DROP CONSTRAINT orders_buyer_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_buyer_id_fkey
    FOREIGN KEY (seller_id) REFERENCES users(id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE orders DROP CONSTRAINT orders_buyer_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_buyer_id_fkey
    FOREIGN KEY (buyer_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE orders DROP CONSTRAINT orders_buyer_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_buyer_id_fkey
    FOREIGN KEY (seller_id) REFERENCES users(id) ON DELETE CASCADE;
