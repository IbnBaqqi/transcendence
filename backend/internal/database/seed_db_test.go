package database_test

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// sql/seed.sql replaced a Go program, so nothing compiles it any more and a
// typo in it would only surface when somebody ran it by hand. This applies the
// real file to a throwaway database and asserts the two things a broken seed
// would still look fine without: that the accounts exist, and that they can
// actually be signed in to.
//
// The file is executed as one string, which is also the check that it stays
// free of psql meta-commands - those would fail here while still working in a
// terminal, and that difference is what lets adminer run it too.
func applySeed(t *testing.T) *database.DB {
	t.Helper()

	db := testdb.New(t)

	seed, err := os.ReadFile("../../sql/seed.sql")
	if err != nil {
		t.Fatalf("reading the seed file: %v", err)
	}
	if _, err := db.ExecContext(t.Context(), string(seed)); err != nil {
		t.Fatalf("applying sql/seed.sql: %v", err)
	}
	return db
}

func TestSeedFillsTheDatabase(t *testing.T) {
	db := applySeed(t)

	counts := []struct {
		name  string
		query string
		want  int
	}{
		{"users", `SELECT count(*) FROM users`, 21},
		{"admins", `SELECT count(*) FROM users WHERE role = 'ADMIN'`, 1},
		{"profiles", `SELECT count(*) FROM profiles`, 21},
		{"listings", `SELECT count(*) FROM listings`, 50},
		{"sellers with listings", `SELECT count(DISTINCT seller_id) FROM listings`, 20},
	}

	for _, c := range counts {
		var got int
		if err := db.QueryRow(c.query).Scan(&got); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s = %d, want %d", c.name, got, c.want)
		}
	}
}

// The password column used to hold a literal placeholder, which no bcrypt
// comparison can match - every account existed and none could be signed in to.
// One password across all of them is deliberate, so this checks both ends of
// the range as well as the admin.
func TestSeededAccountsCanSignIn(t *testing.T) {
	db := applySeed(t)

	const password = "admin123"
	for _, email := range []string{
		"admin@metsatori.com",
		"forager01@example.com",
		"forager20@example.com",
	} {
		t.Run(email, func(t *testing.T) {
			var hash string
			if err := db.QueryRow(
				`SELECT password FROM users WHERE email = $1`, email,
			).Scan(&hash); err != nil {
				t.Fatalf("not seeded: %v", err)
			}
			if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
				t.Errorf("the stored password does not verify against %q: %v", password, err)
			}
		})
	}
}

// lpad, not format('%02s'), because Postgres pads a width with spaces - which
// put a space inside the address for foragers 1 to 9 and made them unloginable
// by anyone typing the obvious thing.
func TestSeededEmailsAreZeroPadded(t *testing.T) {
	db := applySeed(t)

	var n int
	if err := db.QueryRow(
		`SELECT count(*) FROM users WHERE email LIKE '% %' OR username LIKE '% %'`,
	).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d seeded accounts have a space in the email or username", n)
	}
}

func TestSeedIsRerunnable(t *testing.T) {
	db := applySeed(t)

	seed, err := os.ReadFile("../../sql/seed.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(t.Context(), string(seed)); err != nil {
		t.Fatalf("second run failed - the seed truncates first, so it must be repeatable: %v", err)
	}

	var users int
	if err := db.QueryRow(`SELECT count(*) FROM users`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 21 {
		t.Errorf("users = %d after two runs, want 21", users)
	}
}
