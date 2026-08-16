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
- **The timestamp is deliberately coarse.** `last_seen_at` is truncated to the
  minute before it leaves the process. Since it's only written once a minute, the
  extra precision was never information — but it would have been a
  microsecond-accurate activity log of when someone is at their computer.

**What it adds.** Followers are what make a marketplace repeatable rather than
transactional: a buyer can find the forager they trusted last autumn, and a
seller can build an audience. It is also the substrate the notification and
recommendation work builds on later.

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
