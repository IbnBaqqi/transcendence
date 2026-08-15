-- name: GetAddress :one
SELECT * FROM addresses
WHERE user_id = $1
LIMIT 1;

-- name: UpsertAddress :one
INSERT INTO addresses (user_id, location)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
SET location   = EXCLUDED.location,
  updated_at = CURRENT_TIMESTAMP
RETURNING *;
