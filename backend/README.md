# transcendence

transcendence is a Go backend service.

## Prerequisites

- Go 1.26.3+
- PostgreSQL
- [goose](https://github.com/pressly/goose) for migrations


## Setup

1. Clone the repository:
```bash
git clone github.com/IbnBaqqi/transcendence
cd transcendence
```

2. Install dependencies:
```bash
go mod download
```

3. Copy environment file:
```bash
cp .env.example .env
```

4. Update `.env` with your configuration

5. Run migrations:
```bash
make migrate-up
```


## Running

Development mode with hot reload:
```bash
make dev
```

Standard run:
```bash
make run
```

Build binary:
```bash
make build
./bin/transcendence
```

## Environment Variables

See `.env.example` for all available configuration options.

## Project Structure

```
.
├── cmd/
│   └── server/          # Application entrypoint
├── internal/
│   ├── app/             # Application setup
│   ├── config/          # Configuration
│   ├── database/        # Database layer
│   ├── handler/         # HTTP handlers
│   ├── logger/          # Logging
│   └── middleware/      # HTTP middleware
└── sql/
    ├── migrations/      # Database migrations
    └── queries/         # SQL queries for sqlc
```

## Development

Run tests:
```bash
make test
```

Format code:
```bash
make fmt
```

Generate sqlc code:
```bash
make sqlc
```


## License

MIT
