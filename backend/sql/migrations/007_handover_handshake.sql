-- +goose Up

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS seller_handed_over_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS buyer_received_at     TIMESTAMP;


UPDATE orders SET status = 'confirmed' WHERE status = 'paid';

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'confirmed', 'completed', 'cancelled'));

-- +goose Down

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'confirmed', 'paid', 'completed', 'cancelled'));

ALTER TABLE orders
    DROP COLUMN IF EXISTS seller_handed_over_at,
    DROP COLUMN IF EXISTS buyer_received_at;
