-- +goose Up

-- Unusable since 016 moved the stuck rule into the admin_orders view: the
-- planner sees one opaque COALESCE(handshake_stuck OR stranded, false) and has
-- no way to match a partial index on the predicate's old shape. It was still
-- maintained on every write to orders, so this is cost with no query behind it.
-- 017's orders_created_at_idx is what serves the admin list now.
DROP INDEX IF EXISTS orders_stuck_idx;

-- +goose Down

-- Recreated exactly as 015 wrote it, so a rollback restores what was there
-- rather than something close to it.
CREATE INDEX orders_stuck_idx ON orders (created_at DESC)
    WHERE status = 'confirmed'
      AND (seller_handed_over_at IS NULL) <> (buyer_received_at IS NULL);
