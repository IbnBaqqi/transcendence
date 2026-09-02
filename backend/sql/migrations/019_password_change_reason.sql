-- +goose Up
ALTER TABLE refresh_tokens DROP CONSTRAINT refresh_tokens_revoked_reason_check;
ALTER TABLE refresh_tokens ADD CONSTRAINT refresh_tokens_revoked_reason_check
    CHECK (revoked_reason IN ('rotated', 'logout', 'password_reset', 'password_change'));

-- +goose Down
ALTER TABLE refresh_tokens DROP CONSTRAINT refresh_tokens_revoked_reason_check;
UPDATE refresh_tokens SET revoked_reason = 'logout' WHERE revoked_reason = 'password_change';
ALTER TABLE refresh_tokens ADD CONSTRAINT refresh_tokens_revoked_reason_check
    CHECK (revoked_reason IN ('rotated', 'logout', 'password_reset'));
