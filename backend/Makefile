-include .env
export

# ...but NOT the test database URL. `export` above would hand it to every
# recipe, so `make test` would silently run the database tests instead of
# skipping them — and fail on a machine with no Postgres. test-db passes it
# explicitly instead.
unexport TEST_DB_URL

.PHONY: run build test test-db test-race test-coverage clean dev lint sqlc fmt deps \
        migrate-up migrate-down migrate-status seed db-reset demo \
        docker-build docker-up docker-down docker-logs

# Run the application
run:
	@go run ./cmd/server

# Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@air

# Build the application
build:
	@echo "Building..."
	@go build -o bin/transcendence ./cmd/server
	@echo "✓ Binary built: bin/transcendence"

# Run tests
test:
	@go test -v ./...

# Run every test, including the ones that need Postgres.
# TEST_DB_URL is separate from DB_URL on purpose: these tests CREATE and DROP
# databases, and pointing them at a development database would be destructive.
# The URL names the `postgres` maintenance database — each test makes its own.
test-db:
	$(require-test-db-url)
	@TEST_DB_URL="$(TEST_DB_URL)" go test -count=1 -v ./...

# Same as test-db but with the race detector, which is what CI runs. The
# notification dispatcher is the only concurrent code here, and its test proves
# nothing without this.
test-race:
	$(require-test-db-url)
	@TEST_DB_URL="$(TEST_DB_URL)" go test -race -count=1 ./...

# Run tests with coverage
test-coverage:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

# Clean build artifacts
clean:
	@rm -f bin/transcendence
	@rm -f coverage.out
	@echo "✓ Cleaned"

# Generate sqlc code
sqlc:
	@sqlc generate
	@echo "✓ Generated sqlc code"

# Format code
fmt:
	@go fmt ./...
	@echo "✓ Formatted code"

# Lint code
lint:
	@golangci-lint run

# Install dependencies
deps:
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies updated"

# Migrations
# .env is optional (-include at the top), so every goose target checks DB_URL
# itself - otherwise goose gets an empty connection string and fails with a
# confusing driver error instead of saying what's actually missing.
require-db-url = @test -n "$(DB_URL)" || { echo "DB_URL not set - copy .env.example to .env"; exit 1; }

# Same idea for the test database: without it `go test` silently SKIPS every
# database test and still prints ok, which reads like success.
require-test-db-url = @test -n "$(TEST_DB_URL)" || { echo "TEST_DB_URL not set - see .env.example"; exit 1; }

migrate-up:
	$(require-db-url)
	@goose -dir sql/migrations postgres "$(DB_URL)" up

# Wipes the database and rebuilds it from the baseline. Needed after a schema
# change goose cannot migrate into - it only counts versions, so a database on
# older numbering sees the baseline, reports "no migrations to run", and keeps
# the old schema.
#
# Drops the database rather than the volume: `docker compose down -v` would
# take the uploads and the Go module cache with it, and the volume's name
# depends on the compose project name.
db-reset:
	$(require-db-url)
	@docker compose up -d db
	@docker compose exec -T db psql -U $(POSTGRES_USER) -d postgres \
		-c "DROP DATABASE IF EXISTS $(POSTGRES_DB) WITH (FORCE)"
	@docker compose exec -T db psql -U $(POSTGRES_USER) -d postgres \
		-c "CREATE DATABASE $(POSTGRES_DB)"
	@$(MAKE) --no-print-directory migrate-up
	@$(MAKE) --no-print-directory seed

# Fills an empty database with a few users and listings to click around.
# Truncates first, so it is safe to re-run.
seed:
	$(require-db-url)
	@go run ./cmd/seed

migrate-down:
	$(require-db-url)
	@goose -dir sql/migrations postgres "$(DB_URL)" down

migrate-status:
	$(require-db-url)
	@goose -dir sql/migrations postgres "$(DB_URL)" status

# Docker
docker-build:
	@docker build -t transcendence .

docker-up:
	@docker compose up -d

docker-down:
	@docker compose down

docker-logs:
	@docker compose logs -f

# Runs the API-key demo end to end against a running stack.
demo:
	@../scripts/api-demo.sh
