-- +goose Up
CREATE TABLE password_reset_tokens (
    token_hash text PRIMARY KEY,
    user_id    uuid NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT password_reset_tokens_user_id_fkey FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens (user_id);

ALTER TABLE refresh_tokens DROP CONSTRAINT refresh_tokens_revoked_reason_check;
ALTER TABLE refresh_tokens ADD CONSTRAINT refresh_tokens_revoked_reason_check
    CHECK (revoked_reason IN ('rotated', 'logout', 'password_reset'));

-- +goose Down
DROP TABLE IF EXISTS password_reset_tokens;

ALTER TABLE refresh_tokens DROP CONSTRAINT refresh_tokens_revoked_reason_check;
UPDATE refresh_tokens SET revoked_reason = 'logout' WHERE revoked_reason = 'password_reset';
ALTER TABLE refresh_tokens ADD CONSTRAINT refresh_tokens_revoked_reason_check
    CHECK (revoked_reason IN ('rotated', 'logout'));