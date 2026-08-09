-- name: GetUser :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (username, email, password) 
VALUES (
	$1, $2, $3
)
RETURNING *;

-- name: UpdateUser :exec
UPDATE users
SET username = $2,
	email = $3,
	updated_at = now()
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: TouchLastSeen :exec
UPDATE users
SET last_seen_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateShowOnlineStatus :one
UPDATE users
SET show_online_status = $2,
  updated_at = now()
WHERE id = $1
RETURNING *;
