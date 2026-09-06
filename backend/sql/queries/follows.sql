-- name: FollowUser :execrows
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
    profiles.avatar_filename,
    -- Plain, because the WHERE below now drops a blocked person from the list
    -- outright: hiding their presence was the older, narrower rule.
    users.show_online_status
FROM follows
JOIN users ON users.id = follows.followee_id
-- LEFT: an inner join would silently drop anyone without a profile row.
LEFT JOIN profiles ON profiles.id = users.id
WHERE follows.follower_id = sqlc.arg(subject_id)
  AND users.is_visible
  -- Symmetric, and it removes the row rather than dimming it: a block hides
  -- the two of you from each other everywhere else, and a name left in a list
  -- you cannot open is the inconsistency.
  AND NOT EXISTS (
      SELECT 1 FROM blocks b
      WHERE (b.blocker_id = sqlc.arg(viewer_id) AND b.blocked_id = users.id)
         OR (b.blocker_id = users.id AND b.blocked_id = sqlc.arg(viewer_id))
  )
ORDER BY users.username;

-- name: ListFollowers :many
SELECT
    users.id,
    users.username,
    users.last_seen_at,
    profiles.avatar_filename,
    -- Plain, because the WHERE below now drops a blocked person from the list
    -- outright: hiding their presence was the older, narrower rule.
    users.show_online_status
FROM follows
JOIN users ON users.id = follows.follower_id
-- LEFT: an inner join would silently drop anyone without a profile row.
LEFT JOIN profiles ON profiles.id = users.id
WHERE follows.followee_id = sqlc.arg(subject_id)
  AND users.is_visible
  -- Symmetric, and it removes the row rather than dimming it: a block hides
  -- the two of you from each other everywhere else, and a name left in a list
  -- you cannot open is the inconsistency.
  AND NOT EXISTS (
      SELECT 1 FROM blocks b
      WHERE (b.blocker_id = sqlc.arg(viewer_id) AND b.blocked_id = users.id)
         OR (b.blocker_id = users.id AND b.blocked_id = sqlc.arg(viewer_id))
  )
ORDER BY users.username;

-- name: DeleteFollowsForUser :exec
DELETE FROM follows WHERE follower_id = $1 OR followee_id = $1;
