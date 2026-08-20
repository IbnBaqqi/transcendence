-- name: CreateAPIKey :one
INSERT INTO api_keys (user_id, name, key_hash, key_prefix)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: FindLiveKeyByHash :one
SELECT * FROM api_keys
WHERE key_hash = $1
    AND revoked_at IS NULL;

-- name: ListAPIKeysForUser :many
SELECT id, user_id, name, key_prefix, last_used_at, revoked_at, created_at
FROM api_keys
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: RevokeAPIKey :execrows
UPDATE api_keys
SET revoked_at = now()
WHERE id = $1
    AND user_id = $2
    AND revoked_at IS NULL;

-- name: TouchAPIKey :exec
UPDATE api_keys
SET last_used_at = now()
WHERE id = $1;
