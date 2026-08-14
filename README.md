# transcendence

A marketplace for foraged goods — berries, mushrooms, herbs. Sellers post
listings, buyers browse and message them, with a simulated order flow and no
real money. Built for the 42 `ft_transcendence` project.

React + TypeScript frontend, Go backend, PostgreSQL, all runnable with
`docker compose up`.

## Running the tests

```bash
cd backend
make test        # everything that runs without a database
make test-db     # the above, plus the tests that talk to Postgres
```

`make test-db` needs the database up:

```bash
docker compose up -d db
```

**Tests that need Postgres are skipped unless `TEST_DB_URL` is set.** That keeps
`make test` green on a machine without Docker, but it means a skipped test still
prints `ok` — so use `make test-db` when you want to know the database paths
actually pass. It reads `TEST_DB_URL` from `backend/.env` (copy
`backend/.env.example`) and fails with a clear message if it isn't set.

Each of those tests gets **its own database**, created fresh, migrated from
`sql/migrations`, and dropped when the test finishes. Nothing is shared between
tests, and the schema is always whatever the migrations produce rather than a
separate copy that can drift.

For that reason `TEST_DB_URL` points at the `postgres` maintenance database and
**must never point at a database you develop against** — the harness creates and
drops databases through it.
