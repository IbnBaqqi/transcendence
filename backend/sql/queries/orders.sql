-- name: CreateOrder :one
INSERT INTO orders (listing_id, buyer_id, seller_id, quantity, unit_price, total_price, listing_title)
VALUES ($1, $2, $3, $4, $5, $5::numeric * $4::integer, $6)
RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders
WHERE id = $1;

-- name: ListOrdersForUser :many
SELECT * FROM orders
WHERE buyer_id = $1 OR seller_id = $1
ORDER BY created_at DESC;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetOrderForUpdate :one
SELECT * FROM orders
WHERE id = $1
FOR UPDATE;

-- name: CountOrdersForListing :one
SELECT COUNT(*) FROM orders
WHERE listing_id = $1;

-- name: MarkOrderHandedOver :one
UPDATE orders
SET seller_handed_over_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: MarkOrderReceived :one
UPDATE orders
SET buyer_received_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;
