-- name: CreateListingImage :one
INSERT INTO listing_images (listing_id, filename, position)
VALUES (
    $1,
    $2,
    COALESCE((SELECT MAX(position) + 1 FROM listing_images WHERE listing_id = $1), 0)
)
RETURNING *;

-- name: ListListingImages :many
SELECT * FROM listing_images
WHERE listing_id = $1
ORDER BY position, id;

-- name: CountListingImages :one
SELECT COUNT(*) FROM listing_images
WHERE listing_id = $1;

-- name: DeleteListingImage :one
DELETE FROM listing_images
WHERE id = $1 AND listing_id = $2
RETURNING filename;

-- name: DeleteImagesForListing :many
DELETE FROM listing_images
WHERE listing_id = $1
RETURNING filename;

-- name: ListImagesForListings :many
SELECT * FROM listing_images
WHERE listing_id = ANY(sqlc.arg(listing_ids)::int[])
ORDER BY listing_id, position, id;