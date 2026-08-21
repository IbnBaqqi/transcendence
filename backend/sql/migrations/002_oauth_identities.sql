-- +goose Up
CREATE TABLE oauth_identities (
    provider         text NOT NULL,
    provider_user_id text NOT NULL,
    user_id          uuid NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (provider, provider_user_id),

    CONSTRAINT oauth_identities_user_provider_uq UNIQUE (user_id, provider),

    CONSTRAINT oauth_identities_provider_check
        CHECK (provider IN ('google', 'github')),

    CONSTRAINT oauth_identities_user_id_fkey FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_oauth_identities_user_id ON oauth_identities (user_id);

ALTER TABLE users ALTER COLUMN password DROP NOT NULL;

-- +goose Down
ALTER TABLE users ALTER COLUMN password SET NOT NULL;
DROP TABLE IF EXISTS oauth_identities;
