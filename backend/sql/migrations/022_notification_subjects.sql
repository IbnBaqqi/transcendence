-- +goose Up
-- The kinds this adds point at things that are neither an order nor a
-- conversation: a follower is a person, a removed or sold-out listing is a
-- listing. Both need a subject column of their own.
ALTER TABLE notifications ADD COLUMN actor_id uuid;
ALTER TABLE notifications ADD COLUMN listing_id uuid;

ALTER TABLE notifications ADD CONSTRAINT notifications_actor_id_fkey
    FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE notifications ADD CONSTRAINT notifications_listing_id_fkey
    FOREIGN KEY (listing_id) REFERENCES listings(id) ON DELETE CASCADE;

-- Still exactly one subject, now out of four. This is the rule that guarantees
-- the UI always has somewhere to send the reader, so it is widened rather than
-- dropped.
ALTER TABLE notifications DROP CONSTRAINT notifications_one_subject_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_one_subject_check
    CHECK (num_nonnulls(order_id, conversation_id, actor_id, listing_id) = 1);

-- A follow has no listing to name. The column stays a snapshot where it
-- applies - a listing can be removed and the notification still has to read
-- sensibly - but it cannot be required of every kind any more.
ALTER TABLE notifications ALTER COLUMN listing_title DROP NOT NULL;

ALTER TABLE notifications DROP CONSTRAINT notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN (
    'order_placed',
    'order_handed_over',
    'order_cancelled',
    'order_resolved',
    'chat_request',
    'order_confirmed',
    'order_completed',
    'review_received',
    'new_follower',
    'listing_removed',
    'saved_listing_gone'
));

-- +goose Down
-- The new kinds are deleted rather than mapped: none of the five original ones
-- describes a follow or a moderator removal, and a row that lies about what
-- happened is worse than one that is gone. They exist only because of this
-- migration.
DELETE FROM notifications WHERE kind IN (
    'order_confirmed', 'order_completed', 'review_received',
    'new_follower', 'listing_removed', 'saved_listing_gone'
);

ALTER TABLE notifications DROP CONSTRAINT notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN (
    'order_placed',
    'order_handed_over',
    'order_cancelled',
    'order_resolved',
    'chat_request'
));

-- Any row still carrying a null title predates nothing - the five original
-- kinds all have one - but the column cannot go back to NOT NULL while one
-- exists.
DELETE FROM notifications WHERE listing_title IS NULL;
ALTER TABLE notifications ALTER COLUMN listing_title SET NOT NULL;

ALTER TABLE notifications DROP CONSTRAINT notifications_one_subject_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_one_subject_check
    CHECK (num_nonnulls(order_id, conversation_id) = 1);

ALTER TABLE notifications DROP CONSTRAINT notifications_listing_id_fkey;
ALTER TABLE notifications DROP CONSTRAINT notifications_actor_id_fkey;
ALTER TABLE notifications DROP COLUMN listing_id;
ALTER TABLE notifications DROP COLUMN actor_id;
