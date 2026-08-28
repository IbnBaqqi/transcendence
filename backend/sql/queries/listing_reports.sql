-- name: CreateReport :exec
INSERT INTO listing_reports (id, listing_id, reporter_id, reason, detail)
VALUES ($1, $2, $3, $4, $5);
-- name: DetachReporter :exec
-- The FK is ON DELETE SET NULL, but that only fires on a real row delete -
-- which account deletion deliberately never does. Detaching has to be explicit.
UPDATE listing_reports SET reporter_id = NULL WHERE reporter_id = sqlc.arg(user_id)::uuid;
