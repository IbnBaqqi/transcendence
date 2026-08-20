# transcendence

A marketplace for foraged goods — berries, mushrooms, herbs. Sellers post
listings, buyers browse and message them, with a simulated order flow and no
real money. Built for the 42 `ft_transcendence` project.

React + TypeScript frontend, Go backend, PostgreSQL, all runnable with
`docker compose up`.

## Modules

The subject requires a written justification for the modules we claim —
why this one, what was technically hard, and what it adds. One section each.

### Friends system → follows + online status

**What the subject asks for.** Two majors mention it:

> *Allow users to interact* — "A friends system (add/remove friends, see friends list)"
>
> *Standard user management* — "Users can add other users as friends and see their online status"

**What we built.** A **directed follow graph** rather than mutual friendship:
`POST`/`DELETE /users/{id}/follow`, `GET /me/following` (the friends list),
`GET /users/{id}/followers`, each row carrying live online status.

**Why follows rather than mutual friendship.** The subject's module list
predates the open-ended web-app version of the project — several modules still
assume a Pong game — so "friends system" is legacy wording for *a social graph
between users*. Ours is shaped for a marketplace, where the relationship is
genuinely asymmetric: a buyer follows a seller they bought from, and a seller
accumulates followers without having to approve anyone. Requiring a mutual
handshake for that would be worse product design, not better compliance. The
capabilities the subject names — add, remove, see the list, see online status —
are all present.

**What was technically interesting.**

- **The data model is the design.** `follows` has no surrogate id and no status
  column: the composite primary key `(follower_id, followee_id)` is
  simultaneously the row's identity, the guarantee that you cannot follow
  someone twice, and the index for "who does this user follow?". A `CHECK`
  constraint makes self-following impossible at the database level rather than
  by a handler remembering to look. The reverse question needs its own index,
  because a composite key can only be searched from its leading column.
- **Errors are classified rather than leaked.** Following a user who does not
  exist is a foreign key violation; unhandled, that reaches the client as
  `500 something went wrong`. We map it to 404, alongside the equivalent
  handling for duplicate-key violations elsewhere in the codebase.
- **Online status without WebSockets.** Middleware stamps `last_seen_at` at most
  once a minute, and a user counts as online if that stamp is inside a two-minute
  window — twice the write interval, so one missed request doesn't flicker them
  offline. Users can switch the signal off, and when they do the response is
  byte-identical to a user who has never been seen: presence is hidden by
  *absence*, so it cannot be inferred from the shape of the reply.

**What it adds.** Followers are what make a marketplace repeatable rather than
transactional: a buyer can find the forager they trusted last autumn, and a
seller can build an audience. It is also the substrate the notification and
recommendation work builds on later.

### Order lifecycle

**What we built.** A finite state machine over `pending → confirmed →
completed`, with `cancelled` reachable from the first two, plus a full audit
trail of every transition.

**Why it isn't a status column with extra steps.** The subject warns that
trivial implementations of a module are rejected, and "an order has a status
field" would deserve that. Four things make this a lifecycle:

- **The transition table is the design.** Each action declares what states it
  may run from, what it moves to, who is allowed to trigger it, and whether it
  returns stock — as data, in one place, rather than as conditionals scattered
  across four handlers. Adding a state means adding a row.
- **Completion needs both parties.** The seller marks handover and the buyer
  confirms receipt; whichever happens first stamps its own timestamp and leaves
  the status alone. Only the second one completes the order. One person cannot
  declare a transaction finished, which is the whole difficulty of an
  offline-handover marketplace.
- **Stock is part of the transaction.** Placing an order decrements the
  listing's quantity in the same transaction that creates the order, with the
  row locked, so two buyers racing for the last kilo cannot both succeed —
  the loser gets a 409 rather than an oversold listing. Cancelling puts the
  stock back, and is refused once either side has marked the handshake.
- **History is recorded, not inferred.** Every transition writes a row to
  `order_events` — from, to, who, when — inside the transaction that performs
  it. A refused transition leaves nothing behind, and a recorded one cannot be
  a lie. `GET /orders/{id}/events` returns that timeline to the two people
  involved.

**What was technically interesting.**

- **The two commit points.** A handshake mark that doesn't complete the order
  commits early. An event write added only to the status-change path would
  silently lose every "seller marked handover" still waiting on the buyer — so
  the mark is recorded before the branch, which means two events can share one
  transaction, and therefore one `created_at`. The timeline query breaks that
  tie on `id`.
- **Proving the coupling.** Tests that only check "the event was written" pass
  even if the write happens *after* the commit. The test that matters renames
  the events table away, then asserts the **status** didn't change either —
  which is the only way to distinguish "same transaction" from "next to each
  other".
- **Orders outlive what they refer to.** The listing title is snapshotted at
  purchase, so an order still reads correctly after the listing is edited; the
  listing itself cannot be deleted while orders reference it; and neither can
  the buyer or the seller, whose foreign keys are `ON DELETE RESTRICT`. A
  history that either party can erase is not evidence — and the first draft of
  that migration silently dropped one of the two keys, so there is a test
  asserting both deletes are refused.

**What it adds.** Disputes become answerable. "Who cancelled this, and when?"
has a row to point at rather than a word in a column — which is what a
marketplace needs the moment two people disagree about what happened.
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
	r.Use(mw.RequiredAuth)                                       // 401 if not logged in
	r.Use(mw.RequireRole(appService.DB.Queries, auth.RoleAdmin)) // 403 if not an admin
	// admin routes here
})
```

Order matters: `RequiredAuth` first, so "not logged in" answers 401 and "logged
in, wrong role" answers 403.

Roles are matched exactly, not ranked — an `ADMIN` does **not** satisfy
`RequireRole(auth.RoleUser)`. For "any logged-in user", `RequiredAuth` alone is
the check.

## API keys and rate limiting

Machine clients authenticate with a key instead of a login. Create one while
logged in:

```bash
curl -X POST http://localhost:8080/api/v1/me/api-keys \
  -H "Authorization: Bearer <access token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "ci pipeline"}'
```

The response contains the key **once**:

```json
{ "id": 1, "name": "ci pipeline", "key_prefix": "fk_live_a3f9",
  "key": "fk_live_a3f9c2...", "created_at": "..." }
```

Only a SHA-256 hash is stored, so nothing can show it again — if it is lost,
revoke it and create another. Then use it:

```bash
curl http://localhost:8080/api/v1/listings -H "X-API-Key: fk_live_a3f9c2..."
```

A key **acts as its owner**: same permissions, same ownership checks. It cannot
manage keys, though — creating, listing and revoking need a session, so a
leaked key cannot mint a replacement and survive its own revocation.

Revocation takes effect on the next request; the key is looked up every time,
so there is no cache to wait out.

`make demo` (from `backend/`) runs all of this end to end against a running
stack: signup, key creation, the four verbs, the 403 a key gets when it tries
to mint another key, the 429, and the 401 after revocation. It needs a limit
above 6 to reach the throttling section — start the backend with
`RATE_LIMIT_PER_MINUTE=10` to see it trip quickly.

### Limits

Key-authenticated requests are limited to `RATE_LIMIT_PER_MINUTE` (default 60)
per key — per **key**, not per IP, so one noisy client cannot throttle everyone
behind the same NAT. Browser sessions are not limited.

Every response carries `X-RateLimit-Remaining`; a refusal is a **429** with
`Retry-After` in seconds.

**The counters live in memory.** They reset when the API restarts, and each
instance counts separately — two instances behind a load balancer give one key
double its limit. That is fine for this project, and a shared store is what
production would need instead.

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
