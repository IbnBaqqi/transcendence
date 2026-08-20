-- +goose Up
CREATE TABLE IF NOT EXISTS api_keys (
    id           serial PRIMARY KEY,
    user_id      uuid NOT NULL,
    name         text NOT NULL,
    key_hash     text NOT NULL,
    key_prefix   text NOT NULL,
    last_used_at timestamptz,
    revoked_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT api_keys_user_id_fkey FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT api_keys_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT api_keys_key_hash_uq UNIQUE (key_hash)
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);

-- +goose Down
DROP TABLE IF EXISTS api_keys;
