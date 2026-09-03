-- +goose Up

CREATE TABLE notifications (
    id              uuid PRIMARY KEY,
    user_id         uuid NOT NULL,
    kind            text NOT NULL,
    -- Snapshot, like orders.listing_title and conversations.listing_title: a
    -- listing can be removed and the notification still has to read sensibly.
    listing_title   text NOT NULL,
    order_id        uuid,
    conversation_id uuid,
    read_at         timestamptz,
    created_at      timestamptz DEFAULT now(),

    CONSTRAINT notifications_kind_check CHECK (kind IN (
        'order_placed',
        'order_handed_over',
        'order_cancelled',
        'order_resolved',
        'chat_request'
    )),

    -- Every kind links somewhere, and to exactly one thing: the four order
    -- kinds to an order, chat_request to a conversation. Without this a row
    -- can be written that the UI has nowhere to send anyone.
    CONSTRAINT notifications_one_subject_check
        CHECK (num_nonnulls(order_id, conversation_id) = 1),

    CONSTRAINT notifications_user_id_fkey FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT notifications_order_id_fkey FOREIGN KEY (order_id)
        REFERENCES orders(id) ON DELETE CASCADE,
    CONSTRAINT notifications_conversation_id_fkey FOREIGN KEY (conversation_id)
        REFERENCES conversations(id) ON DELETE CASCADE
);

-- Serves both the list and the unread count: the same user's rows, newest
-- first, which is the only way this table is ever read.
CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC);

-- +goose Down

DROP TABLE notifications;
