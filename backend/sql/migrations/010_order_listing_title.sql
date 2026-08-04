-- +goose Up
ALTER TABLE orders ADD COLUMN listing_title text;

UPDATE orders o SET listing_title = l.title
FROM listings l WHERE l.id = o.listing_id;

ALTER TABLE orders ALTER COLUMN listing_title SET NOT NULL;

-- +goose Down
ALTER TABLE orders DROP COLUMN listing_title;