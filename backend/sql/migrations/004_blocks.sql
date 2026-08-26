-- +goose Up
CREATE TABLE blocks (
    blocker_id uuid NOT NULL,
    blocked_id uuid NOT NULL,
    created_at timestamptz DEFAULT now(),

    PRIMARY KEY (blocker_id, blocked_id),

    CONSTRAINT blocks_no_self_block CHECK (blocker_id <> blocked_id),

    CONSTRAINT blocks_blocker_id_fkey FOREIGN KEY (blocker_id)
        REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT blocks_blocked_id_fkey FOREIGN KEY (blocked_id)
        REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS blocks;
