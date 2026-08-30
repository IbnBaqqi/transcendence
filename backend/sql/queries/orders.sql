-- name: CreateOrder :one
INSERT INTO orders (id, listing_id, buyer_id, seller_id, quantity, unit_price, total_price, listing_title)
VALUES ($1, $2, $3, $4, $5, $6, $6::numeric * $5::integer, $7)
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

-- name: ListOrdersForAdmin :many
SELECT * FROM orders
WHERE (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(created_from)::timestamptz IS NULL OR created_at >= sqlc.narg(created_from)::timestamptz)
  AND (sqlc.narg(created_to)::timestamptz IS NULL OR created_at < sqlc.narg(created_to)::timestamptz)
  AND (
      sqlc.narg(stuck)::boolean IS NULL
      OR (status = 'confirmed'
          AND (seller_handed_over_at IS NULL) <> (buyer_received_at IS NULL)
          AND COALESCE(seller_handed_over_at, buyer_received_at)
              < sqlc.arg(stuck_before)::timestamptz) = sqlc.narg(stuck)::boolean
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountOrdersForAdmin :one
SELECT COUNT(*) FROM orders
WHERE (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(created_from)::timestamptz IS NULL OR created_at >= sqlc.narg(created_from)::timestamptz)
  AND (sqlc.narg(created_to)::timestamptz IS NULL OR created_at < sqlc.narg(created_to)::timestamptz)
  AND (
      sqlc.narg(stuck)::boolean IS NULL
      OR (status = 'confirmed'
          AND (seller_handed_over_at IS NULL) <> (buyer_received_at IS NULL)
          AND COALESCE(seller_handed_over_at, buyer_received_at)
              < sqlc.arg(stuck_before)::timestamptz) = sqlc.narg(stuck)::boolean
  );
