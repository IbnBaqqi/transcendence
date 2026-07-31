-- name: CreateOrder :one
INSERT INTO orders (listing_id, buyer_id, seller_id, quantity, unit_price, total_price)
VALUES ($1, $2, $3, $4, $5, $5::numeric * $4::integer)
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

-- Two separate queries rather than one parameterised by column name: sqlc
-- generates from static SQL, so a column can't be a parameter.

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
