-- name: ListReportedListings :many
SELECT
    l.id AS listing_id,
    l.title,
    l.seller_id,
    l.removed_at,
    count(*) AS report_count,
    min(r.created_at)::timestamptz AS first_reported_at
FROM listing_reports r
JOIN listings l ON l.id = r.listing_id
WHERE r.status = 'open'
GROUP BY l.id, l.title, l.seller_id, l.removed_at
ORDER BY min(r.created_at);

-- name: ListReportsForListing :many
SELECT id, reporter_id, reason, detail, status, created_at
FROM listing_reports
WHERE listing_id = $1
ORDER BY created_at DESC;

-- name: ResolveOpenReports :execrows
UPDATE listing_reports
SET status = $2
WHERE listing_id = $1 AND status = 'open';

-- name: SetListingRemoved :one
UPDATE listings
SET removed_at = now(),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: RestoreListing :one
UPDATE listings
SET removed_at = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: CreateModerationAction :one
INSERT INTO moderation_actions (id, listing_id, moderator_id, action, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListModerationActions :many
SELECT id, listing_id, moderator_id, action, note, created_at
FROM moderation_actions
WHERE listing_id = $1
ORDER BY created_at DESC;
