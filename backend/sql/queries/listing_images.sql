-- name: CreateListingImage :one
INSERT INTO listing_images (id, listing_id, filename, position)
VALUES (
    $1,
    $2,
    $3,
    COALESCE((SELECT MAX(position) + 1 FROM listing_images WHERE listing_id = $2), 0)
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
WHERE listing_id = ANY(sqlc.arg(listing_ids)::uuid[])
ORDER BY listing_id, position, id;

-- name: SetListingImagePositions :execrows
-- One statement for the whole gallery. WITH ORDINALITY numbers the array as it
-- arrives, so the caller sends the ids in the wanted order and the positions
-- are derived here rather than passed alongside - two parallel arrays could
-- disagree, one cannot.
--
-- The listing_id in the WHERE is what stops an id from another listing being
-- renumbered; :execrows is how the caller checks every id it sent matched a row.
UPDATE listing_images AS li
SET position = v.ord - 1
FROM unnest(sqlc.arg(image_ids)::uuid[]) WITH ORDINALITY AS v(id, ord)
WHERE li.id = v.id AND li.listing_id = sqlc.arg(listing_id);
