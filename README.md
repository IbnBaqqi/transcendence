# transcendence

A marketplace for foraged goods — berries, mushrooms, herbs. Sellers post
listings, buyers browse and message them, with a simulated order flow and no
real money. Built for the 42 `ft_transcendence` project.

React + TypeScript frontend, Go backend, PostgreSQL, all runnable with
`docker compose up`.

## API documentation

The API describes itself. With the stack up (`docker compose up`):

- **<http://localhost:8080/api/docs>** — Swagger UI: every endpoint, its
  request and response shapes, and a "Try it out" button that issues real
  requests. Paste an access token from `POST /auth/login` into **Authorize**
  to call the endpoints that need one.
- **<http://localhost:8080/api/openapi.yaml>** — the raw OpenAPI 3.1 document,
  if you'd rather point a client generator or another tool at it.

Both are served straight out of the binary — the spec and Swagger UI's assets
are embedded with `go:embed`, so there is nothing to install and no CDN
involved.

### Keeping it honest

The spec is hand-written and lives at `backend/api/openapi.yaml`, so **it is
part of the change, not a follow-up**: adding an endpoint means adding it there
in the same pull request.

That isn't left to memory. `TestSpecMatchesRouter` asks chi for its real route
table and compares it against the spec, failing when an endpoint is in one and
not the other. It checks **method and path only** — the request and response
schemas are still on you to keep true:

```
routed but not documented: GET /api/v1/me/profile (add it to api/openapi.yaml)
documented but not routed: PUT /api/v1/me/settings (stale entry in api/openapi.yaml)
```

It needs no database, so `make test` and CI both run it.

Note that `go:embed` bakes the spec in **at build time** — after editing the
YAML, rebuild (`docker compose up --build`) before the change shows up at
`/api/docs`.

## Roles and the first admin

Accounts are `USER` or `ADMIN` — those two values only, enforced by a `CHECK`
constraint, so a typo is rejected rather than stored:

```
ERROR: new row for relation "users" violates check constraint "users_role_check"
```

There is no UI for promotion and no seeded admin account. Everyone signs up as
`USER`; the first admin is made by hand, on purpose:

```bash
docker compose exec db psql -U postgres -d transcendence \
  -c "UPDATE users SET role = 'ADMIN' WHERE email = 'you@example.test';"
```

`UPDATE 1` means it worked; `UPDATE 0` means no account has that email. Adjust
`-U` and `-d` if you changed `POSTGRES_USER` / `POSTGRES_DB` in `.env`.

The role is read from the database on every request that needs it, not from the
token, so **promotion and demotion take effect on the next request** — no
logging out and back in. That is also what makes revoking an admin meaningful:
were the role taken from the JWT, a demoted admin would keep their powers until
the token expired.

### Protecting a route

```go
r.Group(func(r chi.Router) {
	r.Use(mw.RequiredAuth)                                     // 401 if not logged in
	r.Use(mw.RequireRole(appService.DB.Queries, mw.RoleAdmin)) // 403 if not an admin
	// admin routes here
})
```

Order matters: `RequiredAuth` first, so "not logged in" answers 401 and "logged
in, wrong role" answers 403.

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
