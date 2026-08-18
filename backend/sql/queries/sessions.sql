-- name: StoreSession :exec
INSERT INTO refresh_tokens (token_hash, user_id, expires_at)
VALUES ($1, $2, $3);

-- name: FindLiveSession :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1
    AND expires_at > now()
    AND (revoked_at IS NULL OR revoked_at > sqlc.arg(revoked_after));

-- name: RevokeSession :execrows
UPDATE refresh_tokens
SET revoked_at = now()
WHERE token_hash = $1
    AND revoked_at IS NULL;

-- name: RevokeSessionsForUser :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE user_id = $1
    AND revoked_at IS NULL;
