-- name: SaveListing :exec
-- ON CONFLICT DO NOTHING makes this idempotent: saving the same listing twice
-- is a no-op instead of a unique-violation error. The (user_id, listing_id)
-- named in the conflict clause is the composite PK from migration 009.
INSERT INTO saved_listings (user_id, listing_id)
VALUES ($1, $2)
ON CONFLICT (user_id, listing_id) DO NOTHING;

-- name: UnsaveListing :execrows
-- :execrows (not :exec) so Go gets the rows-aaffected count back and can tell
-- "removed it" (1) from "there was nothing removed" (0) -> honest 404.
DELETE FROM saved_listings
WHERE user_id = $1 AND listing_id = $2;

-- name: ListSaveListings :many
-- Joins the link table back to listings and selects l.* so sqlc reuses the
-- existing Listing struct - no new type, and dtos.ToListingResponses still
-- applies. Order by when it was SAVED (s.created_at), not when the listing
-- was posted (l.created_at) - that's what makes it read like a wishlist.
SELECT l.* FROM saved_listings s
JOIN listings l ON l.id = s.listing_id
WHERE s.user_id = $1
ORDER BY s.created_at DESC;