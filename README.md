# transcendence

A marketplace for foraged goods — berries, mushrooms, herbs. Sellers post
listings, buyers browse and message them, with a simulated order flow and no
real money. Built for the 42 `ft_transcendence` project.

React + TypeScript frontend, Go backend, PostgreSQL, all runnable with
`docker compose up`.

## Modules

The subject requires a written justification for the modules we claim —
why this one, what was technically hard, and what it adds. One section each.

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
  the buyer or the seller. A history that either party can erase is not
  evidence.

**What it adds.** Disputes become answerable. "Who cancelled this, and when?"
has a row to point at rather than a word in a column — which is what a
marketplace needs the moment two people disagree about what happened.

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
