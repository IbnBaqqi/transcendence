-- name: CreateOrderEvent :exec
INSERT INTO order_events (order_id, actor_id, from_status, to_status, note)
VALUES (sqlc.arg(order_id), sqlc.arg(actor_id), sqlc.narg(from_status), sqlc.arg(to_status), sqlc.narg(note));

-- name: ListOrderEvents :many
SELECT id, order_id, actor_id, from_status, to_status, note, created_at
FROM order_events
WHERE order_id = $1
ORDER BY created_at, id;
