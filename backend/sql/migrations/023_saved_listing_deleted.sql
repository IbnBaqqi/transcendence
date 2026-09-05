-- +goose Up
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
    'saved_listing_gone',
    'saved_listing_deleted'
));

-- +goose Down
-- The rows are deleted rather than folded into saved_listing_gone: that kind
-- points at a listing, and these exist precisely because theirs is gone.
DELETE FROM notifications WHERE kind = 'saved_listing_deleted';

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
