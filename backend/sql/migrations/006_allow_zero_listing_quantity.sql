-- +goose Up

ALTER TABLE listings DROP CONSTRAINT listings_quantity_check;
ALTER TABLE listings ADD CONSTRAINT listings_quantity_check CHECk (quantity >= 0);

-- +goose Down

ALTER TABLE listings DROP CONSTRAINT listings_quantity_check;
ALTER TABLE listings ADD CONSTRAINT listings_quantity_check CHECk (quantity > 0);
