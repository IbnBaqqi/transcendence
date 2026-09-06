-- name: SaveListing :exec
INSERT INTO saved_listings (user_id, listing_id)
VALUES ($1, $2)
ON CONFLICT (user_id, listing_id) DO NOTHING;

-- name: UnsaveListing :execrows
DELETE FROM saved_listings
WHERE user_id = $1 AND listing_id = $2;

-- name: ListSavedListings :many
SELECT l.* FROM saved_listings s
JOIN listings l ON l.id = s.listing_id
WHERE s.user_id = $1 AND l.removed_at IS NULL
  AND EXISTS (SELECT 1 FROM users u WHERE u.id = l.seller_id AND u.is_visible)
  -- A block hides the seller entirely, so a listing saved before it must go
  -- with them: otherwise the row is here to click and only the order refuses.
  -- The save itself is kept, so unblocking brings it back.
  AND NOT EXISTS (
      SELECT 1 FROM blocks b
      WHERE (b.blocker_id = $1 AND b.blocked_id = l.seller_id)
         OR (b.blocker_id = l.seller_id AND b.blocked_id = $1)
  )
ORDER BY s.created_at DESC;

-- name: ListSaversOfListing :many
SELECT user_id FROM saved_listings
WHERE listing_id = $1
  AND user_id <> sqlc.arg(except_user);

-- name: DeleteSavedForUser :exec
DELETE FROM saved_listings WHERE user_id = $1;
