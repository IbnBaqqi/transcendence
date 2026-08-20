-- +goose Up
-- Every rune unicode.IsSpace reports, enumerated rather than approximated:
-- plain trim() strips spaces only, and [[:space:]] is locale-dependent and
-- typically ASCII, so an em space or an ideographic space would survive and
-- leave the hole this migration exists to close - a legacy 'aino<em space>'
-- kept, the index built over it, and a fresh signup as 'aino' accepted.
-- Named once here so the four uses below cannot drift apart.
UPDATE users
SET username = btrim(username, ws.chars),
    email    = btrim(email, ws.chars)
FROM (SELECT E'\u0009\u000a\u000b\u000c\u000d\u0020\u0085\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000' AS chars) ws
WHERE username <> btrim(username, ws.chars)
    OR email <> btrim(email, ws.chars);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_uq;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_uq;

CREATE UNIQUE INDEX users_username_lower_uq ON users (lower(username));
CREATE UNIQUE INDEX users_email_lower_uq ON users (lower(email));

-- +goose Down
DROP INDEX IF EXISTS users_username_lower_uq;
DROP INDEX IF EXISTS users_email_lower_uq;

ALTER TABLE users ADD CONSTRAINT users_username_uq UNIQUE (username);
ALTER TABLE users ADD CONSTRAINT users_email_uq UNIQUE (email);
