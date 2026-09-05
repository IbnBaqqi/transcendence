-- Demo data for evaluations: one admin, twenty foragers, fifty listings.
--
-- Not a migration. goose reads sql/migrations and sqlc reads sql/queries, so
-- neither sees this file; it describes content rather than shape, and it is
-- re-runnable because it truncates first.
--
-- Deliberately plain SQL with no psql meta-commands (\i, \set, \copy), so the
-- same file runs three ways:
--   make seed
--   docker compose exec -T db psql -U postgres -d transcendence < backend/sql/seed.sql
--   pasted into adminer  (docker compose --profile tools up -d adminer, :8081)

BEGIN;

-- The whole reason this can be SQL at all. The login path compares with
-- bcrypt, and crypt(..., gen_salt('bf')) is the one thing in Postgres that
-- emits the same $2a$ format - without it every account seeded here would be
-- data to look at rather than one anybody could sign in to.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

TRUNCATE listings, users CASCADE;

-- gen_random_uuid() is v4 where the app mints v7. That is fine for users and
-- listings, which are never ordered by id - but messages and conversations ARE
-- (ORDER BY id, relying on v7 sorting by time), so seeding chat from here would
-- need a v7 function rather than this.
--
-- is_visible is a generated column (deleted_at IS NULL AND suspended_at IS
-- NULL); naming it in an INSERT is an error, so it is absent on purpose.
INSERT INTO users (id, email, username, password, role)
VALUES (
    gen_random_uuid(),
    'admin@metsatori.com',
    'admin',
    crypt('admin123', gen_salt('bf', 10)),
    'ADMIN'
);

-- Same password as the admin, on purpose: one credential to remember when
-- clicking through as different people during an evaluation.
INSERT INTO users (id, email, username, password)
SELECT
    gen_random_uuid(),
    -- lpad, not format('%02s', ...): Postgres pads a width with SPACES, so
    -- %02s puts a space inside the address for the single-digit foragers.
    'forager' || lpad(n::text, 2, '0') || '@example.com',
    'forager-' || lpad(n::text, 2, '0'),
    crypt('admin123', gen_salt('bf', 10))
FROM generate_series(1, 20) AS n;

-- profiles.id IS the user id - there is no user_id column. Signup creates this
-- row, so an account seeded without one has a profile page that cannot load.
INSERT INTO profiles (id) SELECT id FROM users;

-- Fifty listings dealt round-robin: sellers cycle every 20 and titles every 10,
-- so each forager ends up with two or three, and the same goods appear from
-- different people the way a real marketplace looks. Price and quantity are
-- nudged by the row number so no two rows are identical.
WITH sellers AS (
    SELECT id, row_number() OVER (ORDER BY username) AS n
    FROM users
    WHERE role = 'USER'
),
templates (n, title, description, category, price, quantity, unit) AS (
    VALUES
        (1,  'Golden Chanterelles',  'Freshly foraged this morning in the coastal pine forest.', 'mushrooms',  18.00, 4, 'lb'),
        (2,  'Wild Morels',          'Hand-picked from a recent burn site, excellent flavor.',   'mushrooms',  32.00, 2, 'lb'),
        (3,  'Foraged Fiddleheads',  'Tightly curled ostrich fern fiddleheads, spring harvest.', 'greens',      9.50, 6, 'lb'),
        (4,  'Wild Blueberries',     'Small, sweet lowbush blueberries from a sunny hillside.',  'berries',     7.00, 10, 'lb'),
        (5,  'Blackberries',         'Ripe wild blackberries picked along the river trail.',     'berries',     6.50, 8, 'lb'),
        (6,  'Stinging Nettles',     'Young nettle tops, great for soups and teas.',             'greens',      5.00, 5, 'lb'),
        (7,  'Lion''s Mane Mushroom','Found growing on a downed hardwood, very fresh.',          'mushrooms',  22.00, 3, 'lb'),
        (8,  'Wild Ramps',           'Foraged in small batches to keep the patch healthy.',      'vegetables', 15.00, 3, 'bunch'),
        (9,  'Rose Hips',            'Sun-ripened rose hips, hand-cleaned and ready for tea.',   'other',       4.50, 4, 'lb'),
        (10, 'Lingonberries',        'Tart lingonberries picked from an open pine heath.',       'berries',     8.00, 6, 'lb')
)
INSERT INTO listings (id, seller_id, title, description, category, price, quantity, unit)
SELECT
    gen_random_uuid(),
    s.id,
    t.title,
    t.description,
    t.category,
    t.price + ((g.n % 5) * 0.5),
    t.quantity + (g.n % 4),
    t.unit
FROM generate_series(1, 50) AS g(n)
JOIN sellers   s ON s.n = ((g.n - 1) % 20) + 1
JOIN templates t ON t.n = ((g.n - 1) % 10) + 1;

COMMIT;
