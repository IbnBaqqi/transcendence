-- name: CreateReport :exec
INSERT INTO listing_reports (id, listing_id, reporter_id, reason, detail)
VALUES ($1, $2, $3, $4, $5);
