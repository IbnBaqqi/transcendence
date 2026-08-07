-- +goose Up
CREATE TABLE IF NOT EXISTS conversations (
    id serial PRIMARY KEY,
    listing_id integer NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    buyer_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    seller_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'declined')),
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT conversations_listing_buyer_uq UNIQUE (listing_id, buyer_id),
    CONSTRAINT conversations_no_self_chat CHECK (buyer_id <> seller_id)
);

CREATE INDEX idx_conversations_buyer_id ON conversations(buyer_id);
CREATE INDEX idx_conversations_seller_id ON conversations(seller_id);

CREATE TABLE IF NOT EXISTS messages (
    id serial PRIMARY KEY,
    conversation_id integer NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body text NOT NULL CHECK (length(btrim(body)) > 0),
    ready_at timestamp,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);

-- +goose Down
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
