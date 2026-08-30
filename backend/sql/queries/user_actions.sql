-- name: CreateUserAction :one
INSERT INTO user_actions (id, subject_id, moderator_id, action, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListUserActions :many
SELECT * FROM user_actions
WHERE subject_id = $1
ORDER BY created_at DESC;

-- name: DetachUserActionModerator :exec
UPDATE user_actions SET moderator_id = NULL WHERE moderator_id = sqlc.arg(user_id)::uuid;
