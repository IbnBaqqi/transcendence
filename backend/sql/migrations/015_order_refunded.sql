-- +goose Up

ALTER TABLE orders DROP CONSTRAINT orders_status_check;

ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'confirmed', 'completed', 'cancelled', 'refunded'));

CREATE INDEX orders_stuck_idx ON orders (created_at DESC)
    WHERE status = 'confirmed'
      AND (seller_handed_over_at IS NULL) <> (buyer_received_at IS NULL);

-- +goose Down

-- Rolling back loses the refunded/cancelled distinction permanently: the CHECK
-- restored below has no 'refunded', so these rows have to become something it
-- accepts. 'cancelled' asserts the trade never happened, which for an order the
-- seller had already handed over is false. Down is not fidelity-preserving.
UPDATE orders SET status = 'cancelled' WHERE status = 'refunded';

DROP INDEX IF EXISTS orders_stuck_idx;

ALTER TABLE orders DROP CONSTRAINT orders_status_check;

ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'confirmed', 'completed', 'cancelled'));
