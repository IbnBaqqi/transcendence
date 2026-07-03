-- name: CreateListing :one
INSERT INTO listings (seller_id, title, description, category, price, quantity, unit)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetListings :one
SELECT * FROM listings
WHERE id = $1;

-- name: ListListings :many
SELECT * FROM listings
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