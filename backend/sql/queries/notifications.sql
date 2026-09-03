-- name: CreateNotification :exec
INSERT INTO notifications (id, user_id, kind, listing_title, order_id, conversation_id)
VALUES ($1, $2, $3, $4, $5, $6);

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
