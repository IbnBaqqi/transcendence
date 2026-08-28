-- name: BlockUser :exec
INSERT INTO blocks (blocker_id, blocked_id)
VALUES ($1, $2)
ON CONFLICT (blocker_id, blocked_id) DO NOTHING;

-- name: UnblockUser :execrows
DELETE FROM blocks
WHERE blocker_id = $1 AND blocked_id = $2;

-- name: ListBlocks :many
SELECT
    users.id,
    users.username,
    blocks.created_at
FROM blocks
JOIN users ON users.id = blocks.blocked_id
WHERE blocks.blocker_id = $1
ORDER BY users.username;

-- name: BlockExistsBetween :one
SELECT EXISTS (
    SELECT 1
        FROM blocks
    WHERE (blocker_id = sqlc.arg(user_a) AND blocked_id = sqlc.arg(user_b))
        OR (blocker_id = sqlc.arg(user_b) AND blocked_id = sqlc.arg(user_a))
);
-- name: DeleteBlocksForUser :exec
-- Only the blocks this user made. Rows where they are the blocked party belong
-- to someone else: if A blocked B and B then deletes their account, removing
-- A's row would bring B's thread back into A's inbox - a person A deliberately
-- hid, returning as "Deleted user". A departed user cannot act, so being
-- blocked by someone else costs nothing.
DELETE FROM blocks WHERE blocker_id = $1;
