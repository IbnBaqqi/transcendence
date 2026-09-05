-- +goose Up
-- SET NULL rather than RESTRICT so a seller can delete a listing that has
-- already sold. An order does not need its listing: listing_title, unit_price,
-- quantity and total_price are snapshotted on the order row at purchase, which
-- is what lets it read correctly with nothing behind it.
-- conversations.listing_id has been SET NULL since 001 for the same reason.
--
-- Not CASCADE: reviews.order_id is RESTRICT, so cascading listings into orders
-- would abort the delete whenever a buyer had left a review - a condition the
-- seller cannot see - and would let a seller erase a buyer's receipt.
ALTER TABLE orders ALTER COLUMN listing_id DROP NOT NULL;
ALTER TABLE orders DROP CONSTRAINT orders_listing_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_listing_id_fkey
    FOREIGN KEY (listing_id) REFERENCES listings(id) ON DELETE SET NULL;

-- admin_orders selects o.*, and sqlc reads the migrations in order: a view
-- defined back in 016, while listing_id was still NOT NULL, keeps generating a
-- non-nullable column and the admin list would fail to scan the first order
-- whose listing is gone. Recreating it here re-reads the column as it is now.
-- Same definition as 016, moved forward - change both together.
DROP VIEW admin_orders;
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
-- Destructive, and it has to be: an order whose listing is gone cannot satisfy
-- NOT NULL again, and there is no listing to point it back at.
DELETE FROM orders WHERE listing_id IS NULL;
ALTER TABLE orders DROP CONSTRAINT orders_listing_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_listing_id_fkey
    FOREIGN KEY (listing_id) REFERENCES listings(id) ON DELETE RESTRICT;
ALTER TABLE orders ALTER COLUMN listing_id SET NOT NULL;

DROP VIEW admin_orders;
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
