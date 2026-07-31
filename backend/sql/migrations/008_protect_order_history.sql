-- +goose Up

-- An order is a historical record - effectively a receipt. Deleting a listing
-- must never destroy the orders that reference it, but ON DELETE CASCADE (set
-- in 005) does exactly that: one DELETE /listings/{id} silently erases every
-- order for that listing, completed ones included, for BOTH parties.
--
-- RESTRICT flips it: Postgres refuses to delete a listing while any order still
-- references it. DeleteListing checks first and returns a friendly 409, so this
-- constraint is the backstop for any code path that forgets to.
ALTER TABLE orders DROP CONSTRAINT orders_listing_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_listing_id_fkey
    FOREIGN KEY (listing_id) REFERENCES listings(id) ON DELETE RESTRICT;

-- +goose Down

ALTER TABLE orders DROP CONSTRAINT orders_listing_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_listing_id_fkey
    FOREIGN KEY (listing_id) REFERENCES listings(id) ON DELETE CASCADE;
