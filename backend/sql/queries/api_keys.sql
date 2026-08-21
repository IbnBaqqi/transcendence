-- name: CreateKey :one
INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: FindLiveKeyByHash :one
SELECT
    api_keys.id,
    api_keys.user_id,
    users.username,
    users.role
FROM api_keys
JOIN users ON users.id = api_keys.user_id
WHERE api_keys.key_hash = $1
    AND api_keys.revoked_at IS NULL;

-- name: ListKeysForUser :many
SELECT id, user_id, name, key_prefix, last_used_at, revoked_at, created_at
FROM api_keys
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: RevokeKey :execrows
UPDATE api_keys
SET revoked_at = now()
WHERE id = $1
    AND user_id = $2
    AND revoked_at IS NULL;

-- name: TouchKey :exec
UPDATE api_keys
SET last_used_at = now()
WHERE id = $1;
