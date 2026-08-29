-- name: GetUser :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmail :one
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
WHERE id = $1 AND deleted_at IS NULL;

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
SET email              = 'deleted-' || id::text || '@deleted.invalid',
    username           = 'deleted-' || id::text,
    password           = NULL,
    last_seen_at       = NULL,
    show_online_status = false,
    deleted_at         = now(),
    updated_at         = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UserIsActive :one
SELECT EXISTS (
    SELECT 1 FROM users
    WHERE id = $1 AND deleted_at IS NULL AND suspended_at IS NULL
);

-- name: GetSuspension :one
SELECT suspended_at, suspension_reason FROM users
WHERE id = $1;

-- name: SuspendUser :one
UPDATE users
SET suspended_at      = now(),
    suspension_reason = sqlc.arg(reason),
    updated_at        = now()
WHERE id = $1 AND suspended_at IS NULL AND deleted_at IS NULL
RETURNING *;

-- name: ReinstateUser :one
UPDATE users
SET suspended_at      = NULL,
    suspension_reason = NULL,
    updated_at        = now()
WHERE id = $1 AND suspended_at IS NOT NULL AND deleted_at IS NULL
RETURNING *;

-- name: CountAdmins :one
SELECT COUNT(*) FROM users
WHERE role = 'ADMIN' AND deleted_at IS NULL AND suspended_at IS NULL;

-- name: ListUsersForAdmin :many
SELECT id, username, email, role, created_at, last_seen_at,
       suspended_at, suspension_reason, deleted_at
FROM users
WHERE (sqlc.narg(role)::text IS NULL OR role = sqlc.narg(role)::text)
  AND (
      sqlc.narg(status)::text IS NULL
      OR (sqlc.narg(status)::text = 'active'    AND deleted_at IS NULL AND suspended_at IS NULL)
      OR (sqlc.narg(status)::text = 'suspended' AND suspended_at IS NOT NULL AND deleted_at IS NULL)
      OR (sqlc.narg(status)::text = 'deleted'   AND deleted_at IS NOT NULL)
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountUsersForAdmin :one
SELECT COUNT(*) FROM users
WHERE (sqlc.narg(role)::text IS NULL OR role = sqlc.narg(role)::text)
  AND (
      sqlc.narg(status)::text IS NULL
      OR (sqlc.narg(status)::text = 'active'    AND deleted_at IS NULL AND suspended_at IS NULL)
      OR (sqlc.narg(status)::text = 'suspended' AND suspended_at IS NOT NULL AND deleted_at IS NULL)
      OR (sqlc.narg(status)::text = 'deleted'   AND deleted_at IS NOT NULL)
  );
