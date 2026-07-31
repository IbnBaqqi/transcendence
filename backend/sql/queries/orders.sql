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
