-- name: UpsertTag :one
WITH inserted AS (
    INSERT INTO tags (name) VALUES ($1)
    ON CONFLICT (name) DO NOTHING
    RETURNING id
)
SELECT id FROM inserted
UNION ALL
SELECT id FROM tags WHERE name = $1
LIMIT 1;

-- name: AttachTag :exec
INSERT INTO listing_tags (listing_id, tag_id)
VALUES ($1, $2)
ON CONFLICT (listing_id, tag_id) DO NOTHING;

-- name: DetachAllTags :exec
DELETE FROM listing_tags
WHERE listing_id = $1;

-- name: ListTagsForListing :many
SELECT t.name FROM listing_tags lt
JOIN tags t ON t.id = lt.tag_id
WHERE lt.listing_id = $1
ORDER BY t.name;

-- name: ListTagsForListings :many
SELECT lt.listing_id, t.name FROM listing_tags lt
JOIN tags t ON t.id = lt.tag_id
WHERE lt.listing_id = ANY(sqlc.arg(listing_ids)::uuid[])
ORDER BY lt.listing_id, t.name;
