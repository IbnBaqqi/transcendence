-- +goose Up

CREATE TABLE categories (
    slug        text PRIMARY KEY CHECK (slug = lower(slug) AND slug <> '' AND length(slug) <= 50),
    name        text NOT NULL,
    parent_slug text,

    is_top        boolean GENERATED ALWAYS AS (parent_slug IS NULL) STORED,
    parent_is_top boolean GENERATED ALWAYS AS
                      (CASE WHEN parent_slug IS NULL THEN NULL ELSE true END) STORED,

    UNIQUE (slug, is_top),
    FOREIGN KEY (parent_slug, parent_is_top) REFERENCES categories (slug, is_top)
);

INSERT INTO categories (slug, name) VALUES
    ('mushrooms',  'Mushrooms'),
    ('berries',    'Berries'),
    ('greens',     'Greens'),
    ('vegetables', 'Vegetables'),
    ('other',      'Other');

UPDATE listings SET category = lower(btrim(category));
UPDATE listings SET category = 'other'
WHERE category NOT IN (SELECT slug FROM categories);

ALTER TABLE listings ALTER COLUMN category TYPE text;

ALTER TABLE listings
    ADD CONSTRAINT listings_category_fkey
    FOREIGN KEY (category) REFERENCES categories (slug) ON DELETE RESTRICT;

CREATE INDEX listings_category_idx ON listings (category);

-- +goose Down

DROP INDEX IF EXISTS listings_category_idx;
ALTER TABLE listings DROP CONSTRAINT IF EXISTS listings_category_fkey;
ALTER TABLE listings ALTER COLUMN category TYPE varchar(50);
DROP TABLE categories;
