-- +goose Up
CREATE TABLE reviews (
    id          uuid PRIMARY KEY,
    order_id    uuid NOT NULL,
    seller_id   uuid NOT NULL,
    reviewer_id uuid,
    rating      integer NOT NULL,
    comment     varchar(1000),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),

   CONSTRAINT reviews_order_id_uq UNIQUE (order_id),

    CONSTRAINT reviews_rating_check CHECK (rating BETWEEN 1 AND 5),

    CONSTRAINT reviews_order_id_fkey FOREIGN KEY (order_id)
        REFERENCES orders(id) ON DELETE RESTRICT,
    CONSTRAINT reviews_seller_id_fkey FOREIGN KEY (seller_id)
        REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT reviews_reviewer_id_fkey FOREIGN KEY (reviewer_id)
        REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_reviews_seller_id ON reviews (seller_id);

CREATE INDEX idx_reviews_reviewer_id ON reviews (reviewer_id);

-- +goose Down
DROP TABLE IF EXISTS reviews;
