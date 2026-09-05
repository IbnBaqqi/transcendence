# transcendence

A marketplace for foraged goods — berries, mushrooms, herbs. Sellers post
listings, buyers browse and message them, with a simulated order flow and no
real money. Built for the 42 `ft_transcendence` project.

React + TypeScript frontend, Go backend, PostgreSQL, all runnable with
`docker compose up`.

## Running it

```bash
make setup          # .env from the example, and one directory - see below
docker compose up
```

Then <http://localhost:5173> for the app and <http://localhost:8080/api/docs>
for the API. Migrations are a separate step the first time:
`cd backend && make migrate-up`.

**`make setup` before the first `up`, and the order matters.** It creates
`frontend/node_modules` and `backend/uploads` so that Docker does not. A
container cannot mount onto a path that does not exist, so the daemon creates
any missing subdirectory of a bind mount — as **root**, inside your working
tree. Both directories are gitignored, so both are absent on a fresh clone and
both get created that way. After that, the next host-side command that writes
there fails with `EACCES` — `npm ci` for one, `make run` saving an uploaded
image for the other — with an error that blames permissions and never mentions
Docker.

**This is a Linux thing.** Docker Desktop for macOS maps bind-mount ownership to
the host user, so nothing lands root-owned and none of the above happens there —
measured both ways on both platforms. `make setup` is still the right first
command everywhere; `mkdir -p` costs nothing.

Already hit it, on Linux? The directories have to go before they can be
recreated — but **no `sudo` is needed**, because they are empty. Docker's volume
shadows each one, so everything written there goes into the volume rather than
the tree, and removing a directory needs write permission on its *parent*, which
you have:

```bash
rmdir frontend/node_modules backend/uploads
make setup && (cd frontend && npm ci)
```

`rmdir` also fails safely if one of them somehow is not empty, which is the
point at which `sudo rm -rf` becomes the right tool rather than the reflex.

Running the container as your own user does not help — the daemon prepares the
mount point before the container starts, so its user is irrelevant.

### The production-shaped stack, over HTTPS

The dev stack above is plain HTTP and reloads on every edit, which is what you
want while working. To run the thing the way it is meant to be served — both
images built, no source mounts, no toolchain in the containers, everything on
**one HTTPS origin**:

```bash
make setup                                        # includes `make certs`
docker compose -f docker-compose.prod.yml up --build
```

Then <https://localhost>. The API is behind the same origin at
`/api/v1`, so there is no second port and nothing to configure in the browser.
Migrations run themselves here: a one-shot `migrator` service applies them
before the backend starts.

**Your browser will warn about the certificate, and that is expected.**
`make certs` generates a self-signed one — nothing trusts the issuer, because
there is no issuer. Click through it. If you would rather not, `mkcert -install
&& mkcert localhost` writes a locally-trusted certificate to the same two paths
and the warning goes away.

Two things follow from having TLS rather than being incidental to it. The
refresh cookie is `Secure` in this stack, so it is only ever sent over HTTPS —
which is why plain `http://localhost` redirects instead of serving the app; a
session started there would fail to refresh with nothing on screen to say why.
And nginx listens on 8443 inside the container rather than 443, because it runs
as a non-root user that cannot bind a privileged port — compose publishes it as
443, so the privileged bind happens in the daemon.

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

### OAuth login

**What we built.** Sign in with Google or GitHub via the authorization code
flow, linking to an existing account when the email matches.

**What was technically interesting.**

- **The account pre-hijacking defence is the whole feature.** Linking on a
  matching email is the obvious design and it is a takeover: signup does not
  verify addresses, so an attacker registers `victim@gmail.com` with a password
  of their choosing, and when the victim later signs in with Google they are
  silently dropped into the attacker's account. Auto-linking is therefore
  limited to *password-less* rows, which can only have come from this flow;
  anything else is refused and told to sign in with its password first. The
  branch review found this, and the test that pins it is the most valuable one
  in the feature.
- **Emails are only trusted when the provider says they are verified.** GitHub
  lets anyone add any address to their account, so an unverified one proves
  nothing. With no verified address the sign-in is refused rather than
  inventing one, because `users.email` is `NOT NULL` and a synthesised row
  outlives the mistake.
- **`SameSite=Lax` on the state cookie is load-bearing, not a default.**
  `Strict` is the instinct for a CSRF token and it breaks the flow 100% of the
  time: the callback is a top-level cross-site navigation from the provider,
  which is exactly the case `Strict` suppresses. It would fail every sign-in
  with nothing in the logs — so the server now refuses to boot if `PUBLIC_URL`
  and `FRONTEND_URL` cannot deliver that cookie.
- **No access token in the redirect.** A browser redirect can only carry data
  in the URL, and an access token there lands in history, proxy logs and the
  next `Referer`. The callback sets only the refresh cookie and the SPA calls
  the existing `POST /auth/refresh`.
- **One dependency, not two.** Only the root `golang.org/x/oauth2` is imported;
  `oauth2/google` exists mostly for GCE service-account credentials and would
  pull in `cloud.google.com/go`, so the two endpoint URLs are declared by hand.

**PKCE is deliberately not implemented.** It is the first thing an
OAuth-familiar reader looks for, so: PKCE exists to protect *public* clients —
mobile apps and SPAs that cannot hold a secret — where an intercepted
authorization code can be redeemed by whoever intercepts it. This is a
confidential client: the code is exchanged server-to-server with a
`client_secret` the browser never sees, so a stolen code is useless without it.
CSRF, the other thing PKCE is sometimes pressed into covering, is handled by
the `state` cookie. Adding PKCE would not be wrong, and it is what we would add
first if the frontend ever exchanged codes itself — it simply is not what
protects this flow today.

**What it adds.** One fewer password for the user, and one fewer password for
us to store. It also makes `users.password` nullable, which is what lets an
account exist without one at all.

### The backend framework

**What the subject asks for.** The Web category names two separate minors — a
frontend framework and a backend one — and defines a framework as a structured
architecture with conventions, built-in features for common tasks such as
routing, and an ecosystem of tools around it. Its examples are "Express,
Fastify, NestJS, Django, Flask, Ruby on Rails".

**The objection, first.** chi's own README calls it a *router*, and the same
page of the subject lists jQuery and Axios as things that are **not**
frameworks. Anyone who reads one line of chi's documentation has a fair
question, and this section exists to answer it rather than hope it is not
asked.

**Why we claim it anyway.** Express is on the subject's own list, and Express is
a router plus a middleware pipeline over Node's `http`. chi is a router plus a
middleware pipeline over Go's `net/http` — the same relationship, to the same
kind of standard library, solving the same problem. If Express is a framework,
the structural argument that chi is not becomes hard to state.

The definition the subject gives is about **how the application is organised**,
not about what a dependency calls itself. On that reading:

- **Routing with per-group middleware.** `backend/internal/app/router.go` is one
  table: an `/api/v1` route with three groups nested inside it — authenticated,
  then admin-only and session-only within that. Thirteen `r.Use` declarations,
  and a route's protection is a property of where it sits rather than of what
  its handler remembers to check.
- **Built-ins used as built-ins.** Request ids and client-IP extraction come
  from chi itself, which is the "built-in features for common tasks" half of
  the definition doing its job rather than something we had to write.
- **Our own middleware on the same seam**, seven files in
  `backend/internal/middleware/`: structured logging, panic recovery, JWT and
  API-key authentication, role and suspension checks, presence stamping, rate
  limiting and a request-body cap — each written against chi's interface and
  mounted the same way its own are.
- **Layering as a convention, not a suggestion.** Fourteen packages under
  `internal/`, with handler → service → database strictly one-directional:
  handlers parse and respond, services own the rules and the transactions, and
  nothing above `database/` writes SQL.
- **Codegen and migrations as the data-access convention** — sqlc generates the
  query layer from `sql/queries/`, goose versions the schema, and the generated
  Go is never hand-edited.
- **The contract is enforced, not documented.** `backend/api/openapi.yaml` is
  hand-written and embedded in the binary, and a test walks chi's own route
  table against it, failing when a route is in one and not the other.

**Where the line actually is.** jQuery and Axios are libraries you *call* from
code organised some other way — remove Axios and you change call sites. chi is
what the code is organised *inside*: removing it means rewriting the routing
table, re-deriving every group's middleware chain by hand, and finding a new
home for the guarantees that currently come from where a route is declared.
That is the difference the subject's definition is pointing at, and it is the
test we would want applied to any candidate.

**What it adds.** Honestly: no user-visible feature. What it buys is that
adding an endpoint is a one-line declaration in a table whose surrounding group
already decides who may reach it — which is why "is this route inside the
`RequiredAuth` group?" is a question a test can ask, and does.

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

## After the id migration — recreate your database

The migration history was replaced with a single baseline
(`001_initial_schema.sql`) and every id became a uuid. goose tracks versions by
number, so a database already at version 21 sees one file, decides it is
applied, and reports:

```
goose: no migrations to run. current version: 21
```

That leaves the old integer schema in place while the code sends uuids, and the
app half-works before failing on the first listing insert. Recreate it once:

```bash
cd backend && make db-reset
```

That drops and recreates the database, applies the migrations and re-seeds.
Note that `make docker-down` does **not** do this — it deliberately keeps the
volume, so the old schema comes straight back.

## Cleaning up unused tags

A tag is created the first time a listing uses it and is not removed when the
last listing drops it, so the table grows with every new name anyone types.
Collecting the unused ones is a deliberate command rather than something the
listing write path does:

```bash
cd backend && make tag-sweep
```

Nothing schedules it, so tags accumulate until someone runs it.

It is safe against a live database in the sense that matters: it takes the same
advisory lock every tag write takes, so it cannot collect a tag a listing is
being saved with — a sweep without that lock deletes the tag *and* the new
listing's link to it, and neither side reports an error.

It is not free, though. That lock is exclusive and is held for the whole scan,
so every listing save that touches tags waits for the sweep to finish. The scan
is a full pass over `tags`, which is the table that grows without bound — so
this is a maintenance window, not a background job. Run it when a short stall on
listing saves does not matter, and if the table ever gets big enough for that to
hurt, batch the delete with a `LIMIT` and re-take the lock per batch rather than
running the whole thing more often.

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

**This is for the *first* admin only.** After that,
`PATCH /api/v1/admin/users/{id}/role` promotes and demotes from the admin
console, writing a `promoted` or `demoted` row to the account's history. The
psql route stays documented because the bootstrap problem is real: the endpoint
requires an admin, so the very first one has nobody to grant it.

Two refusals are worth knowing. You cannot change your own role — an admin who
demotes themselves loses the endpoint that would undo it. And the last active
admin cannot be demoted: the check takes the roster lock in the same
transaction as the write, so two admins demoting each other at the same moment
cannot both succeed and leave the instance with none.

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
`RATE_LIMIT_PER_MINUTE=10 docker compose up -d backend` to see it trip
quickly. A value below 1 is refused and the default is used instead — it would
mean "no requests at all", not "unlimited".

### Limits

Key-authenticated requests are limited to `RATE_LIMIT_PER_MINUTE` (default 60)
per key — per **key**, not per IP, so one noisy client cannot throttle everyone
behind the same NAT. Browser sessions are not limited.

Key-authenticated responses carry `X-RateLimit-Limit` and
`X-RateLimit-Remaining`; a refusal is a **429** with `Retry-After` in seconds.
Session requests carry neither, because they are not counted.

**The counters live in memory.** They reset when the API restarts, and each
instance counts separately — two instances behind a load balancer give one key
double its limit. That is fine for this project, and a shared store is what
production would need instead.

## Notifications

The app emails people when something happens to them: signup, an order placed,
handed over or cancelled, and a new chat request. Nothing else — emailing on
every create/update/delete would be spam, and it is how a sending domain gets
blacklisted.

### Nothing leaves your machine

`docker compose up` starts [Mailpit](https://mailpit.axllent.org/), which
accepts mail and shows it in a browser instead of delivering it. Read what was
"sent" at **http://localhost:8025**.

Leave `SMTP_HOST` empty and notifications are logged instead, so nobody needs a
mail server to place an order.

### How it works, and what it does not guarantee

Sending happens **after** the database transaction commits, on a worker
goroutine. Two consequences worth knowing:

- **An email failure never fails the action.** The order is what the user asked
  for; the email is a courtesy. A dead relay is a log line.
- **Delivery is at-most-once.** The database and the mail server are two
  systems with no shared transaction, so "row written" and "mail sent" cannot
  be atomic. If the process dies between them the mail is lost — nobody is ever
  emailed about something that did not happen, but a notification can go
  missing. The upgrade path is an outbox table; it is not built, because a
  missed "your order shipped" is a nuisance and an outbox is a table, a worker
  and a migration.

A full queue drops rather than blocks, for the same reason: blocking would put
the mail server back on the request path.

### Why the inbox has more kinds than the mail

The in-app inbox **is** the notification system; email is a courtesy on top of
it. So the inbox grows as the app grows a new thing worth knowing, and the set
of things that send mail does not.

That is deliberate rather than unfinished. Emailing on every action is how a
sending domain gets blacklisted, and most of what belongs in an inbox does not
belong in someone's mail: an order you confirmed, an order that completed.

Two things are absent from the inbox as well, and for the same kind of reason —
the app already tells you, better:

- **A suspension** reaches you as a 403 carrying its reason on your very next
  request. A row saying the same thing is a second, worse copy.
- **A reply in a conversation you already have open** shows as an unread count
  in the chat list, which is where you are looking.

### What the inbox covers

The **Notification system** minor asks for "a complete notification system for
all creation, update, and deletion actions". The inbox is that system, and this
is its coverage — every row written inside the transaction that made the change,
so a notification cannot describe something that did not happen:

| You are told | when | it takes you to |
|---|---|---|
| `order_placed` | someone orders your listing | the order |
| `order_confirmed` | the seller accepts your order | the order |
| `order_handed_over` | the goods change hands | the order |
| `order_completed` | the order finishes | the order |
| `order_cancelled` | either side cancels | the order |
| `order_resolved` | support settles a dispute | the order |
| `chat_request` | someone opens a conversation | the chat |
| `review_received` | a review lands on you | your profile, where it is |
| `new_follower` | someone follows you | their profile |
| `listing_removed` | a moderator removes your listing | the listing |
| `saved_listing_gone` | a listing you saved sells out | the listing |
| `saved_listing_deleted` | a listing you saved is deleted | the seller |

Creation, update and deletion are all represented: an order placed, an order
moving through its states, and a saved listing disappearing.

The last two are one event with two rows because they cannot share a subject.
`notifications.listing_id` is a foreign key that cascades, so a row pointing at
a listing being deleted is erased by that same delete — the row has to point at
the seller instead, who survives. A sold-out listing still exists, so that one
points at the listing.

## Hot reload

Both halves of the stack pick up edits without a restart, whichever way you
started it:

```bash
docker compose up        # frontend via Vite, backend via air
cd backend && make dev   # backend only, on the host
```

Save a `.go` file and the backend rebuilds in a few seconds. `api/openapi.yaml`
counts too — it is `go:embed`-ed into the binary, so editing the spec without
watching it would leave `/api/docs` serving a stale copy with nothing to say
why.

Two things worth knowing. The **first `docker compose up` after pulling this
builds an image** rather than pulling one, because the backend now runs a `dev`
stage from `backend/Dockerfile`; later ups hit the layer cache. And **air is
pinned to one version** in both the Dockerfile and the Makefile, so the
container and the host behave identically — `make dev` runs it through `go run`
rather than from your `PATH`, so there is nothing to install.

The build output goes to `/tmp/air`, deliberately outside the repository: under
compose the source is a bind mount, so a binary written into the tree would
appear on your machine owned by the container's user.

## Browser support

Chrome, Brave and Firefox. Brave is Chromium like Chrome, and the subject's own
list counts Edge, which is also Chromium — so a second Chromium browser is a
browser rather than a reskin. What it adds here is **Shields**, whose cookie and
storage blocking is the most likely thing in these three to break a real flow.

### Minimum versions

Set by Tailwind CSS v4, which compiles to `@property` and `color-mix()`:

| Browser | Minimum |
|---|---|
| Chrome / Brave / Edge | **111** (March 2023) |
| Firefox | **128** (July 2024) |
| Safari | **16.4** (March 2023) — see below |

**Safari is not claimed.** Nobody on the team runs macOS, so it has never been
opened; the row is Tailwind's floor, not a result.

### What was tested, and how

Driven with Playwright at 1280x1000 and 320x700, signed out and signed in, on
**Firefox 155**, **Chromium 153** and **Brave 152** — the last being a real
Brave install, confirmed through `navigator.brave`. Chrome itself is the
day-to-day development browser and shares Brave's and Chromium's engine, but it
is not installed on the machine this pass ran on and was not driven here.

Every check below produced **identical results in all three**, which is the
claim the "consistent UI/UX" requirement is really about:

| Check | Result |
|---|---|
| Sign up, then delete the access token and reload | Session returns through the refresh cookie |
| Modal opens with `role="dialog"` and `aria-modal` | Present |
| Description box grows as you type | 42px to 162px |
| Drag a file onto the photo dropzone | Photo added |
| Brand mark paints in the header | 24x32 box |
| Dark mode through `prefers-color-scheme` | `rgb(22, 23, 29)` |
| Header at 320px, signed out, signed in, as admin | No horizontal overflow |
| Console errors | None |

**Brave was tested with Shields both ways.** Playwright launches Chromium
browsers with `--disable-extensions`, and Shields is an extension — so the
default run may not exercise it at all. Repeating the sign-up and refresh-cookie
flow with extensions kept gives the same result: the `refresh_token` cookie is
set `SameSite=Lax`, path-scoped to `/api/v1/auth`, and survives.

### Known limitations

- **Auto-growing text boxes need a 2026 browser.** The bio and description
  fields use `field-sizing: content`, which MDN puts at Baseline June 2026.
  Below that they fall back to a three-row box with a scrollbar — nothing is
  lost but the growing.
- **Number inputs look different.** Firefox draws the spinner arrows on the
  price fields permanently; Chromium reveals them on hover. Cosmetic, and not
  worth overriding a platform control to hide.
- **Modals do not trap focus.** Tab moves out of an open dialog into the page
  behind it, in every browser. `aria-modal` says otherwise, which is a promise
  the keyboard does not yet keep. Escape and the backdrop both close them.

### Two bugs this found

- **The brand mark did not render in Firefox at all.** It was referenced as
  `<use href="/favicon.svg">` with no fragment, which asks for the target
  document's root element — Chromium supports that, Firefox does not. Below
  `sm` the wordmark is hidden, so on a phone in Firefox the top-left corner was
  empty. The mark now lives in the icon sprite and is referenced by id, like
  every other icon. `iconSprite.test.ts` fails on any fragment-less reference.
- **Modals were not announced as dialogs.** No `role`, no `aria-modal` — found
  because a test driver could not locate the dialog it had just opened.

## Running the tests

```bash
cd backend
make test        # everything that runs without a database
make test-db     # the above, plus the tests that talk to Postgres
make test-race   # test-db with the race detector, which is what CI runs
make lint        # what CI's style and security checks run
```

**Run `make lint` before pushing.** It is the one that catches the failure
you cannot see locally otherwise: `go build` does not compile test files, so a
call site left stale by a merge builds fine and fails CI. `go vet` compiles
them, and `make lint` runs it alongside `gofmt`, `staticcheck` and `gosec`.

Nothing to install — the two linters run through `go run` at the versions
pinned in the Makefile, which are the versions CI uses. An installed binary
would be whatever you happened to install, and a local pass would stop meaning
a CI pass. The first run downloads and builds them, so give it a minute; after
that it is seconds.

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
