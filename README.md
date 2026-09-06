*This project has been created as part of the 42 curriculum by akiiski, krepo and thblack-*

# Metsätori

A marketplace for foraged goods — berries, mushrooms and herbs.

## Description

Finland's everyman's right lets anyone forage on most land, and people routinely
end up with more chanterelles than they can eat. Metsätori is where that surplus
finds a buyer — "tori.fi for forageables".

**No money changes hands on the platform**, by design. An order is a
*reservation*, completed by a two-sided handshake: the seller marks handover, the
buyer marks receipt, and the order closes only when both have. Payment and pickup
are arranged between the two people.

**Key features**

- Listings with photos, categories and tags; search with filters, sorting and pagination
- Reservations with an audit trail, seller/buyer handshake and admin resolution
- Direct messaging, gated on the seller accepting the request
- Follows with online status, blocking, saved listings, seller ratings
- In-app and email notifications
- Admin moderation: reports queue, listing moderation, roles, suspensions
- Public REST API with keys, rate limiting and OpenAPI docs
- English, Finnish and Swedish; OS-driven light and dark themes
- GDPR data export and account deletion, both confirmed by email

## Instructions

### Prerequisites

| Tool | Version | Required? |
|---|---|---|
| Docker + Compose | any current | yes — the only hard requirement |
| `make` | any | yes — wraps the compose and database commands |
| Go | 1.26+ | optional — only to run the backend or its tests on the host |
| Node | 20+ | optional — only to run the frontend or its tests on the host |

### Configuration

Credentials live in a root `.env`, which is gitignored. `make setup` creates it
from the committed `.env.example`; the defaults work as-is for local development.
OAuth and SMTP credentials are blank in the example and optional — sign-in with a
password and the local mail catcher both work without them. The project team can
provide OAuth credentials if required.

### Running it

```bash
make setup          # creates .env from .env.example, plus two directories
docker compose up
```

Frontend at http://localhost:5173, API at http://localhost:8080. Migrations apply
themselves: a goose one-shot container runs before the backend starts.

**Run `make setup` before the first `up`.** Docker creates any missing bind-mount
directory as root inside your working tree, and both `frontend/node_modules` and
`backend/uploads` are gitignored, so they are absent on a fresh clone. Creating
them first means Docker mounts over directories that already exist.

### Running it over HTTPS

The dev stack above is plain HTTP and reloads on every edit. The
production-shaped stack builds both images and serves everything from one HTTPS
origin:

```bash
make setup                                        # includes `make certs`
docker compose -f docker-compose.prod.yml up --build
```

Then https://localhost — the API is behind the same origin at `/api/v1`. The
certificate is self-signed, so the browser will warn; click through, or run
`mkcert -install && mkcert localhost` to write a trusted one to the same paths.

The refresh cookie is `Secure` here, which is why `http://localhost` redirects
rather than serving the app: a session started over HTTP could not refresh.

### Demo data

The schema arrives empty. `cd backend && make seed` loads one admin, twenty
foragers and fifty listings.

| Account | Password |
|---|---|
| `admin@metsatori.com` | `admin123` |
| `forager01@example.com` … `forager20@example.com` | `admin123` |

### Other commands

```bash
cd backend  && make test | test-db | migrate-status | db-reset
cd frontend && npm run test | lint | format:check | build
docker compose --profile tools up      # adds Adminer at http://localhost:8081
```

Mailpit catches all outgoing mail at http://localhost:8025. Nothing leaves your
machine.

## Team Information

<!-- TODO — one row per member: 42 login, role(s) (PO / PM / Tech Lead /
     Developer), and a brief description of their responsibilities. -->

| Member | 42 login | Role(s) | Responsibilities |
|---|---|---|---|
| | | | |
| | | | |
| | | | |
| | | | |
| | | | |

## Project Management

**How the work was organised.** Every change lands as a pull request against
`main`; nobody commits to `main` directly. Work is tracked as GitHub Issues with
a `Backend:` / `Frontend:` / `DevOps:` / `Full-stack:` prefix, and features are
deliberately split into separate backend and frontend issues so two people can
work them in parallel. GitHub Actions runs per-area checks on every PR: frontend
format, lint, i18n and build; backend `gofmt`, `go vet` and the test suite.

<!-- TODO — the subject also asks for:
     - meeting cadence / how tasks were distributed between people
     - communication channels used (Discord, Slack, etc.) -->

## Technical Stack

### Frontend

React 19 with TypeScript, built by Vite. React Router v7 for routing, TanStack
Query for server state, React Hook Form with Zod for forms and validation,
Tailwind CSS v4 for styling, i18next for translations, Axios as the HTTP client.
Tested with Vitest and Testing Library.

### Backend

Go 1.26 with the chi v5 router and its middleware chain. Queries are written by
hand and compiled to type-safe Go by sqlc; schema changes are versioned goose
migrations. Authentication is JWT access tokens plus rotating refresh tokens
stored hashed, with bcrypt for passwords. Logging is `log/slog`. The OpenAPI spec
is served by swgui.

### Database — PostgreSQL 16

Chosen because the data is relational and the invariants are the interesting
part, and we wanted them enforced by the database rather than by application
code: `CHECK` constraints as the real floor under stock levels, composite and
partial indexes for the search and feed queries, a generated column for "deleted
or suspended", and `uuid` v7 primary keys that sort by creation time so message
history paginates on the primary key with no extra index.

### Justification for the major technical choices

- **Go over Node** — static typing and a standard library covering HTTP, crypto
  and TLS without dependencies, which keeps the dependency surface small.
- **sqlc over an ORM** — the SQL is the source of truth and a wrong column name
  is a compile error rather than a runtime one. We do not claim the ORM module.
- **`text` + `CHECK` over native Postgres enums** — adding a value to a native
  enum is a migration that cannot run inside a transaction.
- **`numeric` for money, never a float** — binary floating point cannot represent
  decimal currency exactly.
- **Tailwind with semantic tokens** (`bg-surface`, `text-muted`) rather than raw
  colours, so light and dark themes both work from one class.

## Database Schema

24 tables. Every table has a `uuid` v7 primary key and `created_at timestamptz`;
foreign keys cascade on delete.

| Area | Tables | Key fields and relationships |
|---|---|---|
| **Identity** | `users`, `profiles`, `addresses` | `users`: `email citext UNIQUE`, `username`, `password text NULL` (null for OAuth-only accounts), `role`, `deleted_at`, `suspended_at`, `is_visible` (generated). `profiles` 1:1 → `users`; `addresses` N:1 → `users` |
| **Auth** | `refresh_tokens`, `oauth_identities`, `password_reset_tokens`, `api_keys` | All N:1 → `users`. Tokens stored as hashes with `expires_at` / `revoked_at`; `oauth_identities` unique on (`provider`, `provider_user_id`) |
| **Catalog** | `listings`, `listing_images`, `categories`, `tags`, `listing_tags` | `listings`: `title varchar(100)`, `description text`, `price numeric(10,2)`, `quantity integer CHECK (>= 0)`, `unit varchar(20)`, N:1 → `users`. `listing_images` N:1 → `listings`, ordered by `position`. `categories` is a self-referencing tree; `listing_tags` is an N:M join |
| **Trade** | `orders`, `order_events`, `reviews` | `orders`: `quantity CHECK (> 0)`, `status`, `handed_over_at`, `received_at`, `cancelled_at`; N:1 → `listings` and → `users` (buyer). `order_events` N:1 → `orders`, append-only. `reviews` 1:1 → `orders` with `rating CHECK (1..5)` |
| **Social** | `follows`, `blocks`, `conversations`, `messages`, `saved_listings`, `notifications` | `follows` and `blocks` are unique directed pairs of `users`. `conversations` is one per (`listing`, buyer) with a `status` gating messages; `messages` N:1 → `conversations`, body `varchar(2000)`. `notifications` denormalises `listing_title` so a deleted listing still reads sensibly |
| **Moderation** | `listing_reports`, `moderation_actions`, `user_actions` | `listing_reports` N:1 → `listings` with reason, detail `varchar(500)` and resolution. `moderation_actions` and `user_actions` record one row per admin action with its actor and note |

## Features List

<!-- TODO: fill the "Built by" column. -->

| Feature | What it does | Built by |
|---|---|---|
| Authentication | Signup, login, JWT access tokens, rotating refresh tokens in an HttpOnly cookie, password reset by email | |
| OAuth login | Google and GitHub, linking to an existing account by verified email | |
| Profiles | Bio, contact details, avatar upload with an initials fallback, public profile pages | |
| Listings | Create, edit, delete; photos with drag-to-reorder; categories and tags | |
| Search | Keyword, category, location and price filters; five sort orders; pagination | |
| Orders | Reserve stock, seller handover, buyer receipt, cancellation, full event trail | |
| Messaging | Per-listing threads, seller accepts or declines, unread counts | |
| Follows and presence | Follow sellers, followers and following lists, online status | |
| Blocking | Hides both parties from each other and stops new orders between them | |
| Saved listings | Wishlist with a dedicated page | |
| Reviews | Per-order seller ratings — API complete, not surfaced in the UI | |
| Notifications | In-app inbox with an unread badge, plus email for the events that matter | |
| Admin | Reports queue, listing moderation, roles, suspensions, stuck-order resolution | |
| Public API | API keys, per-key rate limits, OpenAPI spec served from the app | |
| GDPR | JSON data export and account deletion, both confirmed by email | |
| Internationalisation | English, Finnish and Swedish with a language switcher | |
| Design system | 29 reusable components, semantic colour tokens, OS-driven dark mode | |

## Modules

**19 points claimed** — 4 Major (2 pts each) + 11 Minor (1 pt each). 14 required.

| Module | Pts | How it was implemented | Built by |
|---|---|---|---|
| Standard user management | 2 | Profile editing, avatar upload with an initials fallback, follows with online status, public profile pages | <!-- TODO --> |
| User interaction | 2 | Per-listing chat gated on the seller accepting; profile system; follows with both lists; blocking | <!-- TODO --> |
| Advanced permissions | 2 | `RequireRole` middleware, admin user CRUD, role changes, suspensions, role-gated views | <!-- TODO --> |
| Public API | 2 | Hashed API keys, per-key rate limiting, OpenAPI spec at `/api/docs`, GET/POST/PUT/DELETE | <!-- TODO --> |
| Frontend framework | 1 | React 19 + TypeScript + Vite + React Router v7 | <!-- TODO --> |
| Backend framework | 1 | chi v5 — routing plus a composable middleware chain (see below) | <!-- TODO --> |
| Advanced search | 1 | Keyword, category, location and price filters; five sort orders; pagination, all driven from the URL | <!-- TODO --> |
| File upload | 1 | Images with client and server validation, progress indicators, drag-to-reorder, deletion, access control | <!-- TODO --> |
| Notification system | 1 | In-app inbox plus email, covering creation, update and deletion events | <!-- TODO --> |
| OAuth 2.0 | 1 | Google and GitHub, linking to an existing account by verified email | <!-- TODO --> |
| Multiple languages | 1 | English, Finnish and Swedish; switcher in the UI; CI fails on hardcoded strings | <!-- TODO --> |
| Custom design system | 1 | 29 reusable components, semantic colour tokens, typography scale, icon sprite | <!-- TODO --> |
| Additional browsers | 1 | Chrome, Firefox and Brave, driven with Playwright at two viewports, signed in and out. Minimum versions are set by Tailwind v4: Chrome/Brave/Edge 111, Firefox 128 | <!-- TODO --> |
| GDPR compliance | 1 | Data request, JSON export, deletion with confirmation, confirmation emails for both | <!-- TODO --> |
| **Order lifecycle** (module of choice) | 1 | Two-sided handshake with an append-only audit trail (see below) | <!-- TODO --> |

### Backend framework: why chi counts

The subject lists Express as a backend framework. Express is a minimal router plus
a middleware chain — no ORM, no templating, no scaffolding — and chi is the same
shape in Go. What we use it for is that chain: request IDs, structured logging,
panic recovery, a global request-body cap, JWT and API-key authentication,
per-user and per-key rate limiting, role enforcement and last-seen tracking,
composed per route group so `/admin/*` inherits the auth chain and adds a role
check in one line.

### Order lifecycle: justification (module of choice)

**Why we chose it.** The project handles no money by design, so "the order is
complete" cannot be settled by a payment gateway — it has to be modelled.

**What technical challenges it addresses.** Both parties hold half the truth: the
seller knows they handed over, the buyer knows they received. Either alone is a
claim. So an order closes only when both timestamps are set, every transition is
appended to `order_events` with its actor, and an admin can resolve a stuck order
with the reason recorded. Reserving stock is a genuine race as well — a
`SELECT … FOR UPDATE` and a conditional `UPDATE … WHERE quantity >= $2`, with
`CHECK (quantity >= 0)` underneath, and a test that fires two buyers at the last
unit and asserts exactly one wins.

**How it adds value.** It is the trust model the whole marketplace hangs off:
without a payment to settle the transaction, the handshake and its audit trail are
what make a reservation mean anything.

**Why Minor.** It is one coherent subsystem rather than a cross-cutting feature
set, so we claim it at 1 point rather than arguing for 2.

## Individual Contributions

<!-- TODO — one subsection per member. `git shortlog -sne --all` gives the commit
     split, but several people have more than one identity in the history, so
     collapse by person before quoting numbers. -->

### <!-- Member -->

**Implemented:** <!-- specific features, modules or components -->

**Challenges:** <!-- what was hard, and how it was overcome -->

## Resources

- [React](https://react.dev/), [React Router](https://reactrouter.com/) and [TanStack Query](https://tanstack.com/query/latest) documentation
- [chi](https://github.com/go-chi/chi) and Go's `net/http` documentation
- [sqlc](https://docs.sqlc.dev/) and [goose](https://github.com/pressly/goose)
- [PostgreSQL 16 manual](https://www.postgresql.org/docs/16/) — constraints, indexes, `FOR UPDATE`
- [RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749) (OAuth 2.0) and [RFC 7519](https://datatracker.ietf.org/doc/html/rfc7519) (JWT)
- [OWASP Cheat Sheet Series](https://cheatsheetseries.owasp.org/) — authentication, session management, file upload
- [Tailwind CSS v4](https://tailwindcss.com/docs) and [i18next](https://www.i18next.com/)

### Use of AI

<!-- TODO — required by the subject: "a description of how AI was used —
     specifying for which tasks and which parts of the project". Cover which
     tools, which parts of the codebase, what kind of work, and what it was not
     used for. -->
