-- name: CreateListing :one
INSERT INTO listings (seller_id, title, description, category, price, quantity, unit)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetListing :one
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

-- name: SearchListings :many
SELECT listings.*
FROM listings
LEFT JOIN users ON listings.seller_id = users.id
LEFT JOIN profiles ON profiles.id = users.id
WHERE
    (sqlc.narg('keyword')::text IS NULL OR
        listings.title ILIKE '%' || sqlc.narg('keyword')::text || '%' OR
        listings.description ILIKE '%' || sqlc.narg('keyword')::text || '%')
    AND (sqlc.narg('category')::text IS NULL OR listings.category = sqlc.narg('category')::text)
    AND (sqlc.narg('min_price')::numeric IS NULL OR listings.price::numeric >= sqlc.narg('min_price')::numeric)
    AND (sqlc.narg('max_price')::numeric IS NULL OR listings.price::numeric <= sqlc.narg('max_price')::numeric)
    AND (sqlc.narg('location')::text IS NULL OR profiles.location ILIKE '%' || sqlc.narg('location')::text || '%')
ORDER BY listings.created_at DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: CountSearchListings :one
SELECT COUNT(*)
FROM listings
LEFT JOIN users ON listings.seller_id = users.id
LEFT JOIN profiles ON profiles.id = users.id
WHERE
    (sqlc.narg('keyword')::text IS NULL OR
        listings.title ILIKE '%' || sqlc.narg('keyword')::text || '%' OR
        listings.description ILIKE '%' || sqlc.narg('keyword')::text || '%')
    AND (sqlc.narg('category')::text IS NULL OR listings.category = sqlc.narg('category')::text)
    AND (sqlc.narg('min_price')::numeric IS NULL OR listings.price::numeric >= sqlc.narg('min_price')::numeric)
    AND (sqlc.narg('max_price')::numeric IS NULL OR listings.price::numeric <= sqlc.narg('max_price')::numeric)
    AND (sqlc.narg('location')::text IS NULL OR profiles.location ILIKE '%' || sqlc.narg('location')::text || '%');