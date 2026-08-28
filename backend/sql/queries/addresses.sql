-- name: GetAddress :one
SELECT * FROM addresses
WHERE user_id = $1
LIMIT 1;

-- name: UpsertAddress :one
-- The id is only used when the row is new; on conflict it is discarded.
INSERT INTO addresses (id, user_id, location)
VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO UPDATE
SET location   = EXCLUDED.location,
  updated_at = CURRENT_TIMESTAMP
RETURNING *;
-- name: DeleteAddressesForUser :exec
DELETE FROM addresses WHERE user_id = $1;
