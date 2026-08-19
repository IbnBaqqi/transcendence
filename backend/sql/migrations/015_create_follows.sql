-- +goose Up
CREATE TABLE IF NOT EXISTS follows (
    follower_id uuid NOT NULL,
    followee_id uuid NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (follower_id, followee_id),

    CONSTRAINT follows_follower_id_fkey FOREIGN KEY (follower_id)
        REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT follows_followee_id_fkey FOREIGN KEY (followee_id)
        REFERENCES users(id) ON DELETE CASCADE,

    CONSTRAINT follows_no_self_follow CHECK (follower_id <> followee_id)
);

CREATE INDEX idx_follows_followee_id ON follows(followee_id);

-- +goose Down
DROP TABLE IF EXISTS follows;
