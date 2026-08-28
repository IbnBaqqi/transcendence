-- name: FollowUser :exec
INSERT INTO follows (follower_id, followee_id)
VALUES ($1, $2)
ON CONFLICT (follower_id, followee_id) DO NOTHING;

-- name: UnfollowUser :execrows
DELETE FROM follows
WHERE follower_id = $1 AND followee_id = $2;

-- name: ListFollowing :many
SELECT
    users.id,
    users.username,
    users.last_seen_at,
    -- Same rule as the inbox: a block in either direction hides presence.
    COALESCE(users.show_online_status AND NOT EXISTS (
        SELECT 1 FROM blocks b
        WHERE (b.blocker_id = sqlc.arg(viewer_id) AND b.blocked_id = users.id)
           OR (b.blocker_id = users.id AND b.blocked_id = sqlc.arg(viewer_id))
    ), false)::boolean AS show_online_status
FROM follows
JOIN users ON users.id = follows.followee_id
WHERE follows.follower_id = sqlc.arg(subject_id)
ORDER BY users.username;

-- name: ListFollowers :many
SELECT
    users.id,
    users.username,
    users.last_seen_at,
    -- Same rule as the inbox: a block in either direction hides presence.
    COALESCE(users.show_online_status AND NOT EXISTS (
        SELECT 1 FROM blocks b
        WHERE (b.blocker_id = sqlc.arg(viewer_id) AND b.blocked_id = users.id)
           OR (b.blocker_id = users.id AND b.blocked_id = sqlc.arg(viewer_id))
    ), false)::boolean AS show_online_status
FROM follows
JOIN users ON users.id = follows.follower_id
WHERE follows.followee_id = sqlc.arg(subject_id)
ORDER BY users.username;
-- name: DeleteFollowsForUser :exec
DELETE FROM follows WHERE follower_id = $1 OR followee_id = $1;
