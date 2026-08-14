package testdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/pressly/goose/v3"

	"github.com/IbnBaqqi/transcendence/internal/config"
	"github.com/IbnBaqqi/transcendence/internal/database"
)

const EnvURL = "TEST_DB_URL"

// New returns a connection to a fresh database with every migration applied.
func New(t *testing.T) *database.DB {
	t.Helper()

	adminURL := os.Getenv(EnvURL)
	if adminURL == "" {
		t.Skipf("set %s to run database tests", EnvURL)
	}

	admin, err := sql.Open("postgres", adminURL)
	if err != nil {
		t.Fatalf("connecting to %s: %v", EnvURL, err)
	}
	if err := admin.Ping(); err != nil {
		t.Fatalf("no database at %s: %v", EnvURL, err)
	}

	name := "test_" + randomSuffix(t)
	if _, err := admin.Exec("CREATE DATABASE " + pq.QuoteIdentifier(name)); err != nil {
		t.Fatalf("creating %s: %v", name, err)
	}

	t.Cleanup(func() {
		if _, err := admin.Exec("DROP DATABASE " + pq.QuoteIdentifier(name)); err != nil {
			t.Errorf("dropping %s: %v", name, err)
		}
		if err := admin.Close(); err != nil {
			t.Errorf("closing admin connection: %v", err)
		}
	})

	db := open(t, withDatabase(t, adminURL, name))
	migrate(t, db.DB)

	return db
}

// open connects through the app's own Connect, so tests exercise the same
// pool settings the server uses.
func open(t *testing.T, dbURL string) *database.DB {
	t.Helper()

	db, err := database.Connect(context.Background(), &config.DBConfig{
		URL:             dbURL,
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("connecting to the test database: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing the test database: %v", err)
		}
	})

	return db
}

// migrate replays sql/migrations, so the test schema is whatever production
// will actually have — no second copy of the schema to drift.
func migrate(t *testing.T, db *sql.DB) {
	t.Helper()

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}

	goose.SetLogger(goose.NopLogger())

	if err := goose.Up(db, migrationsDir(t)); err != nil {
		t.Fatalf("running migrations: %v", err)
	}
}

// migrationsDir resolves sql/migrations from THIS file's location.
func migrationsDir(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the testdb package on disk")
	}

	return filepath.Join(filepath.Dir(thisFile), "..", "..", "sql", "migrations")
}

// withDatabase swaps the database name in a connection URL, keeping the host,
// credentials and query parameters that came with it.
func withDatabase(t *testing.T, rawURL, name string) string {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parsing %s: %v", EnvURL, err)
	}
	parsed.Path = "/" + name

	return parsed.String()
}

func randomSuffix(t *testing.T) string {
	t.Helper()

	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generating a database name: %v", err)
	}

	return hex.EncodeToString(buf)
}
