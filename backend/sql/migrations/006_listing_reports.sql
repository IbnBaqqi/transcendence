-- +goose Up
CREATE TABLE listing_reports (
    id          uuid PRIMARY KEY,
    listing_id  uuid NOT NULL,
    reporter_id uuid,
    reason      text NOT NULL,
    detail      varchar(500),
    status      text NOT NULL DEFAULT 'open',
    created_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT listing_reports_listing_reporter_uq UNIQUE (listing_id, reporter_id),

    CONSTRAINT listing_reports_reason_check
        CHECK (reason IN ('spam', 'prohibited', 'misleading', 'offensive', 'other')),
    CONSTRAINT listing_reports_status_check
        CHECK (status IN ('open', 'upheld', 'dismissed')),

    CONSTRAINT listing_reports_listing_id_fkey FOREIGN KEY (listing_id)
        REFERENCES listings(id) ON DELETE CASCADE,

    CONSTRAINT listing_reports_reporter_id_fkey FOREIGN KEY (reporter_id)
        REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_listing_reports_open
    ON listing_reports (created_at)
    WHERE status = 'open';

-- +goose Down
DROP TABLE IF EXISTS listing_reports;
