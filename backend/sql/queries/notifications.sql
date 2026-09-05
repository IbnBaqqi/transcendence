-- name: CreateNotification :exec
-- Exactly one of the four subject columns is set, enforced by
-- notifications_one_subject_check: a row the UI cannot route is not worth
-- writing. listing_title is a snapshot where a listing is involved and null
-- where none is - a follow has no listing to name.
INSERT INTO notifications (
    id, user_id, kind, listing_title, order_id, conversation_id, actor_id, listing_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListNotifications :many
SELECT * FROM notifications
WHERE user_id = $1
-- id breaks the tie: created_at defaults to now(), and two rows written in one
-- transaction share it. Ids are time-ordered, so this stays chronological.
ORDER BY created_at DESC, id DESC
LIMIT $2;

-- name: MarkNotificationsRead :execrows
UPDATE notifications
SET read_at = CURRENT_TIMESTAMP
WHERE user_id = $1
  AND read_at IS NULL;
