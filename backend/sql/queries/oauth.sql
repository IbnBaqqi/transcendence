-- name: FindUserByProviderIdentity :one
-- Joined rather than a second lookup: the callback needs the user, and this way
-- the role and username come back fresh rather than cached in the link.
SELECT sqlc.embed(users)
FROM oauth_identities
JOIN users ON users.id = oauth_identities.user_id
WHERE oauth_identities.provider = $1
    AND oauth_identities.provider_user_id = $2;

-- name: LinkIdentity :exec
-- No ON CONFLICT: a duplicate means two callbacks are racing to claim the same
-- provider account, and the 23505 is the right answer rather than something to
-- swallow.
INSERT INTO oauth_identities (provider, provider_user_id, user_id)
VALUES ($1, $2, $3);

-- name: ListProvidersForUser :many
SELECT provider FROM oauth_identities
WHERE user_id = $1
ORDER BY provider;
-- name: DeleteIdentitiesForUser :exec
DELETE FROM oauth_identities WHERE user_id = $1;
