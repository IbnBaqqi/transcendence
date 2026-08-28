-- name: GetUser :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmail :one
-- deleted_at excluded: this is the login and password-reset lookup, and a
-- deleted account must not be findable by the address it used to have. The
-- address is scrubbed anyway, so this is belt and braces - but it is the
-- braces that matter if the scrub ever changes.
SELECT * FROM users
WHERE lower(email) = lower(sqlc.arg(email))
  AND deleted_at IS NULL
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

-- name: AnonymiseUser :one
-- The placeholders embed the id because both columns are NOT NULL under unique
-- indexes on lower() - a fixed string would collide on the second deletion, and
-- case cannot be what separates them. What a client shows is "Deleted user",
-- derived at the DTO boundary; this is only what keeps the row valid.
--
-- .invalid is RFC 2606 reserved, so the address can never route anywhere.
--
-- The deleted_at IS NULL guard is what makes a second delete return no rows
-- rather than re-scrubbing an already anonymous row.
UPDATE users
SET email        = 'deleted-' || id::text || '@deleted.invalid',
    username     = 'deleted-' || id::text,
    password     = NULL,
    last_seen_at = NULL,
    deleted_at   = now(),
    updated_at   = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UserIsActive :one
SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL);
