-- name: GetProfile :one
SELECT * FROM profiles
WHERE id = $1
LIMIT 1;

-- name: ListProfiles :many
SELECT * FROM profiles
ORDER BY id;

-- name: CreateProfile :one
INSERT INTO profiles (id, firstname, lastname, bio, phone_number, date_of_birth)
VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateProfile :one
UPDATE profiles
SET firstname     = $2,
    lastname      = $3,
    bio           = $4,
    phone_number  = $5,
    date_of_birth = $6
WHERE id = $1
RETURNING *;

-- name: DeleteProfile :exec
DELETE FROM profiles
WHERE id = $1;

-- name: EnsureProfile :exec
INSERT INTO profiles (id)
VALUES ($1)
ON CONFLICT (id) DO NOTHING;

-- name: GetProfileForUpdate :one
SELECT * FROM profiles
WHERE id = $1
FOR UPDATE;

-- name: SetAvatar :one
 WITH previous AS (
    SELECT avatar_filename FROM profiles WHERE id = $1
)
UPDATE profiles
SET avatar_filename = $2
FROM previous
WHERE profiles.id = $1
RETURNING previous.avatar_filename;

-- name: ClearAvatar :one
WITH previous AS (
  SELECT avatar_filename FROM profiles WHERE id = $1
)
UPDATE profiles
SET avatar_filename = NULL
FROM previous
WHERE profiles.id = $1
RETURNING previous.avatar_filename;
