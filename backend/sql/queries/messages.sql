-- name: CreateMessage :one
INSERT INTO messages (id, conversation_id, sender_id, body)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListRecentMessages :many
SELECT * FROM messages
WHERE conversation_id = $1
ORDER BY id DESC
LIMIT $2;

-- name: ListMessagesAfter :many
SELECT * FROM messages
WHERE conversation_id = $1
  AND id > sqlc.arg(after_id)
ORDER BY id
LIMIT sqlc.arg(max_rows);

-- name: MarkMessagesRead :execrows
UPDATE messages
SET read_at = CURRENT_TIMESTAMP
WHERE conversation_id = sqlc.arg(conversation_id)
  AND sender_id <> sqlc.arg(reader_id)
  AND read_at IS NULL;

-- name: CountUnreadForUser :one
SELECT COUNT(*) FROM messages m
JOIN conversations c ON c.id = m.conversation_id
WHERE (c.buyer_id = sqlc.arg(user_id) OR c.seller_id = sqlc.arg(user_id))
  AND m.sender_id <> sqlc.arg(user_id)
  AND m.read_at IS NULL
  AND NOT EXISTS (
      SELECT 1
        FROM blocks
       WHERE blocks.blocker_id = sqlc.arg(user_id)
         AND blocks.blocked_id = m.sender_id
  );

-- name: ListMessagesForExport :many
-- Every message in every conversation this account is part of - both sides.
-- The other person's text is already readable in the app, so exporting it
-- discloses nothing new; omitting it would produce half a conversation.
SELECT messages.* FROM messages
JOIN conversations ON conversations.id = messages.conversation_id
WHERE conversations.buyer_id = sqlc.arg(user_id)
   OR conversations.seller_id = sqlc.arg(user_id)
ORDER BY messages.conversation_id, messages.id;
