-- name: CreateReview :one
INSERT INTO reviews (id, order_id, seller_id, reviewer_id, rating, comment)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateReview :one
-- Scoped to the author, so "not yours" and "does not exist" are the same
-- no-rows answer.
-- comment_set carries the difference between "absent" and "explicitly
-- cleared": absent leaves the column alone, so a rating fix cannot silently
-- destroy text the author never touched.
UPDATE reviews
SET rating     = sqlc.arg(rating),
    comment    = CASE WHEN sqlc.arg(comment_set)::boolean
                      THEN sqlc.narg(comment)
                      ELSE comment END,
    updated_at = now()
WHERE id = sqlc.arg(id) AND reviewer_id = sqlc.arg(reviewer_id)::uuid
RETURNING *;

-- name: GetReviewForOrder :one
SELECT * FROM reviews
WHERE order_id = $1;

-- name: ListReviewsForSeller :many
-- LEFT JOIN, not JOIN: reviewer_id is NULL once that account is deleted, and
-- an inner join would drop exactly the reviews the SET NULL preserves.
SELECT
    r.id, r.order_id, r.rating, r.comment, r.created_at, r.updated_at,
    r.reviewer_id,
    u.username   AS reviewer_username,
    u.deleted_at AS reviewer_deleted_at
FROM reviews r
LEFT JOIN users u ON u.id = r.reviewer_id
WHERE r.seller_id = $1
ORDER BY r.created_at DESC;

-- name: SellerRating :one
-- AVG returns numeric, which sqlc types as a string; ::float8 makes it a
-- float64. COALESCE covers a seller with no reviews, where AVG is NULL.
SELECT
    COALESCE(AVG(rating), 0)::float8 AS average,
    COUNT(*)                         AS total
FROM reviews
WHERE seller_id = $1;

-- name: DetachReviewer :exec
-- The FK is ON DELETE SET NULL, but account deletion anonymises rather than
-- deleting, so nothing fires it.
UPDATE reviews SET reviewer_id = NULL WHERE reviewer_id = sqlc.arg(user_id)::uuid;
