-- +goose Up
ALTER TABLE users ADD COLUMN suspended_at timestamptz;

ALTER TABLE users ADD COLUMN suspension_reason varchar(500);

CREATE TABLE user_actions (
    id           uuid PRIMARY KEY,
    subject_id   uuid NOT NULL,
    moderator_id uuid,
    action       text NOT NULL,
    note         varchar(500),
    created_at   timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT user_actions_action_check
        CHECK (action IN ('suspended', 'reinstated', 'deleted')),

    CONSTRAINT user_actions_subject_id_fkey FOREIGN KEY (subject_id)
        REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT user_actions_moderator_id_fkey FOREIGN KEY (moderator_id)
        REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_user_actions_subject ON user_actions (subject_id, created_at DESC);

CREATE INDEX idx_user_actions_moderator ON user_actions (moderator_id);

-- +goose Down
DROP TABLE IF EXISTS user_actions;
ALTER TABLE users DROP COLUMN IF EXISTS suspension_reason;
ALTER TABLE users DROP COLUMN IF EXISTS suspended_at;