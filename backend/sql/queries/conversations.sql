-- name: CreateConversation :one
INSERT INTO conversations (id, listing_id, listing_title, buyer_id, seller_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetConversation :one
SELECT * FROM conversations
WHERE id = $1;

-- name: GetConversationForUpdate :one
SELECT * FROM conversations
WHERE id = $1
FOR UPDATE;

-- name: UpdateConversationStatus :one
UPDATE conversations
SET status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: TouchConversation :exec
UPDATE conversations
SET updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: ListConversationsForUser :many
SELECT
    c.id, c.listing_id, c.listing_title, c.buyer_id, c.seller_id, c.status, c.created_at, c.updated_at,
    u.id                    AS other_user_id,
    u.username              AS other_username,
    u.last_seen_at          AS other_last_seen_at,
    u.show_online_status    AS other_show_online_status,
    COALESCE(lm.body, '')   AS last_message_body,
    lm.created_at           AS last_message_at,
    (SELECT COUNT(*)
          FROM messages m
        WHERE m.conversation_id = c.id
          AND m.sender_id <> sqlc.arg(user_id)
          AND m.read_at IS NULL) AS unread_count
FROM conversations c
JOIN users u ON u.id = CASE WHEN c.buyer_id = sqlc.arg(user_id)
                            THEN c.seller_id
                            ELSE c.buyer_id END
LEFT JOIN LATERAL (
    SELECT body, created_at
      FROM messages
     WHERE conversation_id = c.id
     ORDER BY id DESC
     LIMIT 1
) lm ON TRUE
WHERE (c.buyer_id = sqlc.arg(user_id) OR c.seller_id = sqlc.arg(user_id))
  AND NOT EXISTS (
      SELECT 1
        FROM blocks
        WHERE blocks.blocker_id = sqlc.arg(user_id)
          AND blocks.blocked_id = u.id
  );
