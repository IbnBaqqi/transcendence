-- +goose Up
CREATE TABLE IF NOT EXISTS order_events (
    id serial PRIMARY KEY,
    order_id integer NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
    from_status text,
    to_status text NOT NULL,
    note text,
    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_order_events_order_id ON order_events(order_id, created_at, id);

-- +goose Down
DROP TABLE IF EXISTS order_events;