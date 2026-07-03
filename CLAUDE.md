# CLAUDE.md

Guidance for Claude Code **and** for the team. This is a school project built by
junior developers, so it explains concepts, not just commands. Read it before
starting work.

## What we're building

A marketplace for **foraged goods** (berries, mushrooms, herbs) — "tori.fi for
forageables." Sellers post listings, buyers browse and message them, with a
**simulated** (fake) order/payment flow — no real money. Built for the 42
`ft_transcendence` project.

## Tech stack (decided)

| Layer    | Tech                                                              |
|----------|-------------------------------------------------------------------|
| Frontend | React 19 + TypeScript, Vite, React Router v7, Axios, **Tailwind CSS v4** |
| Backend  | Go 1.26, chi router, slog logging                                 |
| Database | PostgreSQL 16                                                     |
| DB tools | goose (migrations), sqlc (query codegen)                          |
| Auth     | JWT access tokens + stored `refresh_tokens`; hashed+salted passwords |
| Dev      | Docker Compose, Make                                              |

## Repo layout

```
backend/
  cmd/server/       main.go — entry point
  internal/
    app/            router + wiring (router.go is where routes live)
    config/         env-var config loading
    database/       DB connection + sqlc-GENERATED code (don't hand-edit)
    handler/        HTTP handlers
    middleware/     logging, request IDs, panic recovery
  sql/migrations/   database structure, versioned (goose)
  sql/queries/      SQL we run from the app (sqlc)
  Makefile          common commands

frontend/
  src/api/client.ts        Axios instance (talks to the backend)
  src/components/layout/    app shell (Header, Footer, Layout)
  src/pages/               page components
  src/routes/index.tsx     route table (URL -> page)
  src/index.css            Tailwind entry + design tokens
```

## Key concepts (read if migrations/queries are new to you)

The database has two jobs, one tool each.

### Migrations = database structure, versioned (goose)
A `.sql` file describing one change to the DB shape ("create the listings
table"). Each has an Up (apply) and Down (undo); goose runs them in numbered
order. New table → add the next-numbered file in `sql/migrations/` → `make
migrate-up`. **Never edit a migration that's already merged/applied — write a
new one.**

### Queries = the SQL we run, turned into Go (sqlc)
Write SQL in `sql/queries/*.sql`, run `make sqlc`, and it generates type-safe Go
functions into `internal/database/`. **That generated Go is auto-created — don't
hand-edit it;** change the `.sql` and re-run `make sqlc`.

### Adding a backend feature (the normal loop)
1. Migration → `make migrate-up`  2. Queries → `make sqlc`
3. Handler in `internal/handler/`  4. Register route in `internal/app/router.go`
5. `make test`, then `make run`

## Common commands

Backend commands run from inside `backend/`.
```bash
# Backend
make run / make dev / make test
make sqlc                 # regenerate Go DB code after editing sql/queries
make migrate-up / migrate-down / migrate-status
# Frontend (from frontend/)
npm run dev / npm run build / npm run lint
```

## Conventions

- **Money**: integer cents (`price_cents`), never floats. Currency EUR.
- **Primary keys**: `UUID` (`gen_random_uuid()`).
- **DB enums**: `text` + `CHECK` constraints (easy to extend).
- **sqlc is generated**: edit `sql/queries/*.sql`, never `internal/database/*.go`.
- **Styling**: Tailwind. Build UI with the **semantic tokens** (`bg-surface`,
  `text-foreground`, `text-muted`, `border-line`, `text-accent`) — not raw
  `brand-*`/`gray-*` — so light/dark theming works automatically (dark mode is
  OS-driven, no toggle).
- **Git**: work on a feature branch, open a PR to `main` — don't commit to `main`.

## Explaining code (junior-friendly comments)

- When providing code — in chat code blocks **and** when writing files — add
  **simple, junior-level comments** explaining what the parts do: imports, type
  annotations, non-obvious syntax, and framework patterns (React hooks, router).
- Keep them short and beginner-friendly. Assume the reader is a junior developer
  refreshing their knowledge, not an expert.
- These comments are primarily for **learning**. The developer decides
  case-by-case whether a comment is worth keeping as real documentation, so write
  them so they're easy to keep or delete.

## How I like to work on code

- I usually want to write the code myself. Give me the code in chat blocks to
  type in, and don't edit the source files directly unless I ask you to.
- Go one file at a time when building something up.
- Include the junior-level explanatory comments (see above) in those blocks.

## Guidance for Claude

- This is a learning project. **Explain the "why"** briefly so teammates learn.
- Prefer small, reviewable changes that follow existing patterns.
- For DB features, follow the 5-step loop and remind the user to run
  `make migrate-up` / `make sqlc`.
- Don't commit, push, or open PRs unless explicitly asked.
- Ask before introducing new dependencies or frameworks.

## Setup notes / gotchas

- Backend needs a `DB_URL` env var. Create `backend/.env` (an `.env.example`
  doesn't exist yet — being addressed).
- Frontend talks to the backend at `http://localhost:8080` (in `api/client.ts`);
  being moved to an env-based `/api` (#45).
- `backend/docker-compose.yml` has broken YAML and is backend-only — a
  root-level single-command compose is planned.

## Open / not yet decided

- Final module list + the 14th point (see the team's plan).
- Which frontend data-fetching library (TanStack Query suggested, #48).
- Full deployment/HTTPS wiring (single-command compose + reverse proxy).
- `species`/`region`/`friendships` inclusion is tentative pending module lock.
