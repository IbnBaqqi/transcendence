-- name: StoreSession :exec
INSERT INTO refresh_tokens (token_hash, user_id, expires_at)
VALUES ($1, $2, $3);

-- name: FindLiveSession :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1
    AND expires_at > now()
    AND (
        revoked_at IS NULL
        OR (revoked_reason = 'rotated' AND revoked_at > sqlc.arg(revoked_after))
      );

-- name: RevokeSession :execrows
UPDATE refresh_tokens
SET revoked_at = now(),
    revoked_reason = sqlc.arg(reason)
WHERE token_hash = sqlc.arg(token_hash)
    AND revoked_at IS NULL;

-- name: RevokeSessionsForUser :exec
-- Rotated rows are included, not just live ones: FindLiveSession still accepts
-- a rotated token for the length of the grace window, so skipping them would
-- leave a logged-out user redeemable for another 30 seconds.
UPDATE refresh_tokens
SET revoked_at = now(),
    revoked_reason = 'logout'
WHERE user_id = $1
    AND expires_at > now()
    AND (revoked_at IS NULL OR revoked_reason = 'rotated');

-- name: FindSessionByHash :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1;

-- name: DeleteDeadSessionsForUser :exec
-- Sessions that can never be redeemed again: expired, or revoked long enough
-- ago that the grace window has closed. Runs on the user's own rows as they
-- issue a new session, so growth is bounded by active sessions rather than by
-- lifetime logins - no scheduler, and the user_id index already covers it.
DELETE FROM refresh_tokens
WHERE user_id = $1
    AND (
        expires_at < now()
        OR (revoked_at IS NOT NULL AND revoked_at < sqlc.arg(revoked_before))
      );
