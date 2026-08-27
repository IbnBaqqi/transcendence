-- +goose Up
-- Postgres indexes the *referencing* side of a foreign key only if you ask.
-- Without these, deleting a user sequentially scans both tables to find the
-- rows to cascade or null out.

-- blocks' primary key is (blocker_id, blocked_id), which serves lookups by
-- blocker_id but not by blocked_id alone. follows has the mirror index
-- (idx_follows_followee_id); blocks was written from it without one.
CREATE INDEX idx_blocks_blocked_id ON blocks (blocked_id);

-- listing_reports_listing_reporter_uq is (listing_id, reporter_id), so the
-- same gap: no index leads with reporter_id. Moderation wants exactly that
-- lookup - "everything this reporter has ever filed" is how a serial
-- false-reporter is spotted.
CREATE INDEX idx_listing_reports_reporter_id ON listing_reports (reporter_id);

-- +goose Down
DROP INDEX IF EXISTS idx_listing_reports_reporter_id;
DROP INDEX IF EXISTS idx_blocks_blocked_id;
