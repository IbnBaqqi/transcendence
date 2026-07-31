-- +goose Up

-- Payment now happens entirely between buyer and seller, off-platform, so the
-- 'paid' state (which described a payment WE simulated) no longer means
-- anything. Completion becomes a two-sided handshake instead: the order only
-- reaches 'completed' once the seller has marked it handed over AND the buyer
-- has marked it received.

-- Nullable timestamps rather than booleans: NULL = "not yet", a value = "yes,
-- and here is when". Same storage, and you get an audit trail for free.
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS seller_handed_over_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS buyer_received_at     TIMESTAMP;

-- Park any order stuck in the old 'paid' state back at 'confirmed', which is
-- where the handshake now begins. Our table is empty today, but a migration
-- has to be safe on a database that isn't.
UPDATE orders SET status = 'confirmed' WHERE status = 'paid';

-- Retighten the enum so 'paid' can never be written again.
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
