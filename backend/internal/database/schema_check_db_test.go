package database_test

import (
	"context"
	"strings"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func TestAssertSchemaCurrentAcceptsTheBaseline(t *testing.T) {
	db := testdb.New(t)

	if err := database.AssertSchemaCurrent(context.Background(), db.DB); err != nil {
		t.Errorf("a freshly migrated database was refused: %v", err)
	}
}

// The case the guard exists for: a database still on the pre-uuid schema, which
// goose will not migrate because it only counts versions.
func TestAssertSchemaCurrentRefusesAnOldSchema(t *testing.T) {
	db := testdb.New(t)

	// Rebuild listings the way it looked before the baseline. The FKs pointing
	// at it have to go first, and they are not what is being tested.
	for _, stmt := range []string{
		`DROP TABLE order_events, orders, messages, conversations,
		            listing_images, saved_listings, listings CASCADE`,
		`CREATE TABLE listings (id serial PRIMARY KEY, title text NOT NULL)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}

	err := database.AssertSchemaCurrent(context.Background(), db.DB)
	if err == nil {
		t.Fatal("an integer-id schema was accepted")
	}
	// The message has to carry the fix: this fires on someone's machine at
	// startup, where the error text is all they get.
	for _, want := range []string{"integer", "docker compose down -v", "make migrate-up"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q:\n%v", want, err)
		}
	}
}

// An empty database is migrate-up's problem to report, not the guard's.
func TestAssertSchemaCurrentIgnoresAnEmptyDatabase(t *testing.T) {
	db := testdb.New(t)

	if _, err := db.Exec(`DROP TABLE listings CASCADE`); err != nil {
		t.Fatal(err)
	}

	if err := database.AssertSchemaCurrent(context.Background(), db.DB); err != nil {
		t.Errorf("an unmigrated database should pass through, got: %v", err)
	}
}
