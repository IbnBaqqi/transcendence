-- +goose Up

-- The seven days live here rather than in Go so the rule has one home: the
-- admin list, its count and /resolve all read `stuck` from this view instead
-- of re-deriving it. The cost is that tuning the threshold is a migration with
-- CREATE OR REPLACE VIEW rather than an edit to a constant - and the 409 in
-- internal/service/admin_order.go spells "7 days" out for the user, so it is a
-- second copy that has to move with this one.

CREATE VIEW admin_orders AS
SELECT shapes.*,
       COALESCE(shapes.handshake_stuck OR shapes.stranded, false)::boolean AS stuck
FROM (
    SELECT o.*,
           COALESCE(
               o.status = 'confirmed'
               AND (o.seller_handed_over_at IS NULL) <> (o.buyer_received_at IS NULL)
               AND COALESCE(o.seller_handed_over_at, o.buyer_received_at)
                   < now() - interval '7 days',
               false)::boolean AS handshake_stuck,
           COALESCE(
               o.status IN ('pending', 'confirmed')
               AND o.seller_handed_over_at IS NULL
               AND o.buyer_received_at IS NULL
               AND buyer.deleted_at IS NOT NULL
               AND seller.deleted_at IS NOT NULL,
               false)::boolean AS stranded
    FROM orders o
    JOIN users buyer ON buyer.id = o.buyer_id
    JOIN users seller ON seller.id = o.seller_id
) shapes;

-- +goose Down

DROP VIEW IF EXISTS admin_orders;
