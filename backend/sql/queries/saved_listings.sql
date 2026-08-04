-- name: SaveListing :exec
INSERT INTO saved_listings (user_id, listing_id)
VALUES ($1, $2)
ON CONFLICT (user_id, listing_id) DO NOTHING;

-- name: UnsaveListing :execrows
DELETE FROM saved_listings
WHERE user_id = $1 AND listing_id = $2;

-- name: ListSaveListings :many
SELECT l.* FROM saved_listings s
JOIN listings l ON l.id = s.listing_id
WHERE s.user_id = $1
ORDER BY s.created_at DESC;