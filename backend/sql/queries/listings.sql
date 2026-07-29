-- name: CreateListing :one
INSERT INTO listings (seller_id, title, description, category, price, quantity, unit)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetListing :one
SELECT * FROM listings
WHERE id = $1;

-- name: ListListings :many
SELECT * FROM listings
WHERE quantity > 0
ORDER BY created_at DESC;

-- name: UpdateListing :one
UPDATE listings
SET title = $2,
    description = $3,
    category = $4,
    price = $5,
    quantity = $6,
    unit = $7,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteListing :exec
DELETE FROM listings
WHERE id = $1;

-- name: GetListingForUpdate :one
SELECT * FROM listings
WHERE id = $1
FOR UPDATE;

-- name: DecrementListingQuantity :one
UPDATE listings
SET quantity = quantity - $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND quantity >= $2
RETURNING *;