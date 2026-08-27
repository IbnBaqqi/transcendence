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

-- name: DeleteDeadResetTokensForUser :exec
-- Tokens that can never be redeemed again: expired, or already spent. Runs on
-- the user's own rows as they ask for a new link, so growth is bounded by live
-- requests rather than by lifetime resets - no scheduler, and the user_id
-- index already covers it. Mirrors DeleteDeadSessionsForUser.
--
-- Must run AFTER the cooldown check: LastResetRequestAt reads the newest row
-- whether or not it is spent, so deleting first would let anyone bypass the
-- cooldown by spending or expiring their link.
DELETE FROM password_reset_tokens
WHERE user_id = $1
    AND (expires_at < now() OR used_at IS NOT NULL);
