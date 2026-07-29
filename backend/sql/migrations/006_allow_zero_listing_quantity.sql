-- +goose Up

-- A listing's quantity is its whole lifecycle: orders decrement it, and when it
-- reaches 0 the listing is closed for good.
ALTER TABLE listings DROP CONSTRAINT listings_quantity_check;
ALTER TABLE listings ADD CONSTRAINT listings_quantity_check CHECk (quantity >= 0);

-- +goose Down
ALTER TABLE listings DROP CONSTRAINT listings_quantity_check;
ALTER TABLE listings ADD CONSTRAINT listings_quantity_check CHECk (quantity > 0);