-- +goose Up
ALTER TABLE listings ADD COLUMN removed_at timestamptz;

CREATE INDEX idx_listings_visible
    ON listings (created_at DESC) WHERE removed_at IS NULL;

CREATE TABLE moderation_actions (
    id           uuid PRIMARY KEY,
    listing_id   uuid NOT NULL,
    moderator_id uuid,
    action       text NOT NULL,
    note         varchar(500),
    created_at   timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT moderation_actions_action_check
        CHECK (action IN ('removed', 'restored', 'dismissed')),

    CONSTRAINT moderation_actions_listing_id_fkey FOREIGN KEY (listing_id)
        REFERENCES listings(id) ON DELETE CASCADE,

    CONSTRAINT moderation_actions_moderator_id_fkey FOREIGN KEY (moderator_id)
        REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_moderation_actions_listing
    ON moderation_actions (listing_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS moderation_actions;
DROP INDEX IF EXISTS idx_listings_visible;
ALTER TABLE listings DROP COLUMN IF EXISTS removed_at;