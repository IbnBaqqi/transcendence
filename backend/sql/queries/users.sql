-- name: GetUser :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE lower(email) = lower(sqlc.arg(email))
LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (id, username, email, password)
VALUES (
	$1, $2, $3, $4
)
RETURNING *;

-- name: UpdateUser :exec
-- No callers today. Whoever wires "edit profile" must normalise first the way
-- normalizeSignupInput does, or padded and case-variant names get back in
-- through this door.
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

-- name: EmailOrUsernameTaken :one
SELECT
    EXISTS(SELECT 1 FROM users u WHERE lower(u.email) = lower(sqlc.arg(email)))       AS email_taken,
    EXISTS(SELECT 1 FROM users u WHERE lower(u.username) = lower(sqlc.arg(username))) AS username_taken;

-- name: GetUserRole :one
SELECT role FROM users
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password = $2,
    updated_at = now()
WHERE id = $1;

-- name: GetUserForUpdate :one
SELECT * FROM users
WHERE id = $1
FOR UPDATE;
