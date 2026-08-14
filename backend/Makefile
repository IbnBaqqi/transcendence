-include .env
export

.PHONY: run build test test-db clean dev lint

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
	@TEST_DB_URL="$(TEST_DB_URL)" go test -v ./...

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
