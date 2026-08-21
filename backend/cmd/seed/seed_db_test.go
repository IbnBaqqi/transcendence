package main

import (
	"database/sql"
	"os"
	"os/exec"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// The seed is the one insert path no unit test covers, and it broke silently
// during the uuid conversion: CreateUserParams gained an ID field, the seed did
// not set it, and Go's zero value is a legal uuid rather than NULL - so the
// first user inserted as 00000000-... and the second collided on the primary
// key. `go build` cannot catch a missing struct field.
//
// This runs the real binary against a throwaway database and then asserts no
// row anywhere holds the nil uuid.
func TestSeedFillsEveryIDIn(t *testing.T) {
	db, url := testdb.NewWithURL(t)

	cmd := exec.Command("go", "run", ".")
	cmd.Env = append(os.Environ(),
		"DB_URL="+url,
		"JWT_SECRET=seed-test-secret",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("seed failed: %v\n%s", err, out)
	}

	var users, listings int
	if err := db.QueryRow(`SELECT count(*) FROM users`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM listings`).Scan(&listings); err != nil {
		t.Fatal(err)
	}
	t.Logf("seeded %d users and %d listings", users, listings)
	if users == 0 || listings == 0 {
		t.Fatal("the seed inserted nothing - it used to stop after the first user")
	}

	assertNoNilUUIDs(t, db.DB)
}

// assertNoNilUUIDs walks the catalog rather than naming tables, so a table
// added later is covered without anyone remembering this test exists.
func assertNoNilUUIDs(t *testing.T, db *sql.DB) {
	t.Helper()

	rows, err := db.Query(`
		SELECT c.relname, a.attname
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relkind = 'r'
		  AND c.relname <> 'goose_db_version'
		  AND a.attnum > 0 AND NOT a.attisdropped
		  AND format_type(a.atttypid, a.atttypmod) = 'uuid'
		ORDER BY c.relname, a.attnum`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	type col struct{ table, name string }
	var cols []col
	for rows.Next() {
		var c col
		if err := rows.Scan(&c.table, &c.name); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, c)
	}
	if len(cols) == 0 {
		t.Fatal("no uuid columns found - the query is wrong, not the schema")
	}

	for _, c := range cols {
		var n int
		if err := db.QueryRow(
			`SELECT count(*) FROM `+c.table+` WHERE `+c.name+` = $1`, uuid.Nil,
		).Scan(&n); err != nil {
			t.Fatalf("%s.%s: %v", c.table, c.name, err)
		}
		if n != 0 {
			t.Errorf("%s.%s holds %d nil uuid(s)", c.table, c.name, n)
		}
	}
	t.Logf("checked %d uuid columns", len(cols))
}
