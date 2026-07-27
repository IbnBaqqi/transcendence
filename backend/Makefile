include .env
export

.PHONY: run build test clean dev lint

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
migrate-up:
	@goose -dir sql/migrations postgres "$(DB_URL)" up

migrate-down:
	@goose -dir sql/migrations postgres "$(DB_URL)" down

migrate-status:
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
