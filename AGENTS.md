# AGENTS.md

Marketplace for foraged goods. Two apps in one repo, all runnable via `docker compose up`:

- `backend/` — Go 1.26, `chi` router, Postgres via sqlc + goose migrations (API on `:8080`)
- `frontend/` — React 19 + TS + Vite + Tailwind 4 (dev server on `:5173`, proxies `/api` and `/uploads` to the backend)

## Commands

Backend (run from `backend/`):
- `make test` — unit tests only. DB-backed tests **silently skip** (print `ok`) unless `TEST_DB_URL` is set.
- `make test-db` — every test for real. Requires a running Postgres (`docker compose up -d db`) and `TEST_DB_URL` in `backend/.env`; fails loudly if unset.
- `make run`, `make dev` (needs `go install github.com/cosmtrek/air@latest`), `make build`, `make lint` (golangci-lint), `make fmt`, `make sqlc`, `make migrate-up|down|status`.

Frontend (run from `frontend/`):
- `npm run lint`, `npm run format:check` (Prettier), `npm test` (vitest), `npm run build` (`tsc -b && vite build`).
- CI order for frontend is `npm ci` → lint → format:check → test → build; run all five before finishing work.

## Backend gotchas

- **`TEST_DB_URL` vs `DB_URL`**: DB tests each create a fresh database (name `test_<random>`), apply `sql/migrations`, and drop it. `TEST_DB_URL` must point at the `postgres` maintenance database and **never** at a dev database — it's destructive by design. `make test` deliberately unsets it; `make test-db` passes it explicitly.
- **sqlc output is committed**: `backend/internal/database/*.sql.go` is generated from `sql/queries/` (config: `sqlc.yaml`). After changing a query or migration, run `make sqlc`. Don't edit generated files by hand.
- **Migrations**: numbered `001_...sql` files in `backend/sql/migrations/`, applied via goose. The test schema is always exactly what migrations produce (no separate schema copy).
- **Two `.env.example` files must stay in sync**: repo root (used by `docker compose`) and `backend/` (used by host `make run`/`migrate-up`). Root `.env` is gitignored. `DB_URL` is overridden inside compose to use host `db`; `UPLOAD_DIR` similarly.
- Compose runs the backend with plain `go run` (no reload). Hot reload is host-only via `make dev`.
- `go run ./cmd/seed` seeds demo data; in `ENV=dev` it **TRUNCATEs listings/users first** (destructive).

## Frontend gotchas

- Forms are react-hook-form + zod; schemas live in `src/schemas/`, components in `src/components/` (forms / objects / text / layout).
- Routes are declared in `src/routes/index.tsx`. API base is `import.meta.env.VITE_API_URL ?? "/api/v1"` (see `src/api/client.ts`); auth token read from `localStorage.accessToken`.
- `vite.config.ts` proxies `/api` and `/uploads` to `http://backend:8080` — a Docker service name, so it only works inside compose.

## Architecture

- Wiring lives in `backend/internal/app/app.go` (services) and `router.go` (routes). All authenticated endpoints hang under `/api/v1`.
- **No websockets.** "Presence" is DB `last_seen`, updated by `TouchLastSeen` middleware throttled in-memory to 1/min with a 2-min online window (`internal/presence`).
- Uploaded images are stored by the backend under `UPLOAD_DIR` and served at `/uploads/<filename>` (no directory listing).
