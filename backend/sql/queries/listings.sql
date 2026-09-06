-- name: CreateListing :one
INSERT INTO listings (id, seller_id, title, description, category, price, quantity, unit)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetListing :one
SELECT * FROM listings
WHERE listings.id = $1;

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

-- name: IncrementListingQuantity :one
UPDATE listings
SET quantity = quantity + $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: ListListingsForExport :many
-- Everything the seller posted, including sold-out and moderator-removed ones.
-- The buyer-facing filters are deliberately absent: this is the seller's own
-- record of what they wrote, not a shop window.
SELECT * FROM listings
WHERE seller_id = sqlc.arg(seller_id)
ORDER BY created_at;

-- name: ListSellerImageFilenames :many
-- Read before the rows go, so the caller can unlink the files after the commit.
-- listing_images CASCADEs from listings, so the rows need no delete of their own
-- - only the files on disk outlive the transaction.
SELECT i.filename
FROM listing_images i
JOIN listings l ON l.id = i.listing_id
WHERE l.seller_id = $1 AND l.removed_at IS NULL;

-- name: DeleteListingsForSeller :exec
-- removed_at IS NULL is the whole point: listing_reports and moderation_actions
-- both CASCADE from listings, so deleting a moderator-removed one takes the
-- report and the record of the decision with it. Account deletion must not be a
-- way to launder a moderation record, and DeleteListing already refuses the same
-- case for the same reason. Those listings are invisible anyway.
DELETE FROM listings
WHERE seller_id = $1 AND removed_at IS NULL;
