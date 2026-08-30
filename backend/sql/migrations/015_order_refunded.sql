-- +goose Up

ALTER TABLE orders DROP CONSTRAINT orders_status_check;

ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'confirmed', 'completed', 'cancelled', 'refunded'));

CREATE INDEX orders_stuck_idx ON orders (created_at DESC)
    WHERE status = 'confirmed'
      AND (seller_handed_over_at IS NULL) <> (buyer_received_at IS NULL);

-- +goose Down

DELETE FROM order_events WHERE order_id IN (SELECT id FROM orders WHERE status = 'refunded');
DELETE FROM orders WHERE status = 'refunded';

DROP INDEX IF EXISTS orders_stuck_idx;

ALTER TABLE orders DROP CONSTRAINT orders_status_check;

ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'confirmed', 'completed', 'cancelled'));
