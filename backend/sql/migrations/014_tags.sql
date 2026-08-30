-- +goose Up

CREATE TABLE tags (
    id   serial PRIMARY KEY,
    name text NOT NULL UNIQUE
         CHECK (name = lower(btrim(name)) AND name <> '' AND length(name) <= 30)
);

CREATE TABLE listing_tags (
    listing_id uuid NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    tag_id     integer NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (listing_id, tag_id)
);

CREATE INDEX listing_tags_tag_id_idx ON listing_tags (tag_id);

-- +goose Down

DROP TABLE listing_tags;
DROP TABLE tags;
