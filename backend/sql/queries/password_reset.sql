-- name: CreateResetToken :exec
INSERT INTO password_reset_tokens (token_hash, user_id, expires_at)
VALUES ($1, $2, $3);

-- name: FindLiveResetToken :one
SELECT * FROM password_reset_tokens
WHERE token_hash = $1
    AND used_at IS NULL
    AND expires_at > now();

-- name: MarkResetTokenUsed :execrows
UPDATE password_reset_tokens
SET used_at = now()
WHERE token_hash = $1
    AND used_at IS NULL;

-- name: InvalidateResetTokensForUser :exec
UPDATE password_reset_tokens
SET used_at = now()
WHERE user_id = $1
    AND used_at IS NULL;

-- name: LastResetRequestAt :one
SELECT created_at FROM password_reset_tokens
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;