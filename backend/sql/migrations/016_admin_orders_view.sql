-- +goose Up

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

DROP VIEW admin_orders;
