package database_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// The migration's trim class must strip exactly what strings.TrimSpace does.
// A rune Go strips but Postgres keeps survives the cleanup, the unique index is
// built over it, and a fresh signup as the bare name is then accepted - the
// duplicate migration 020 exists to prevent.
//
// The class is read out of the migration rather than repeated here, so this
// cannot pass against a copy that has drifted from what actually runs.
func TestTrimClassMatchesGo(t *testing.T) {
	db := testdb.New(t)

	source, err := os.ReadFile("../../sql/migrations/020_case_insensitive_identities.sql")
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`SELECT (E\'[^\']*\') AS chars`).FindSubmatch(source)
	if match == nil {
		t.Fatal("could not find the trim class in migration 020")
	}
	class := string(match[1])

	trimmed := 0
	for r := rune(0); r <= 0x3000; r++ {
		if !unicode.IsSpace(r) {
			continue
		}
		trimmed++

		padded := "aino" + string(r)
		var got string
		if err := db.QueryRow(fmt.Sprintf(`SELECT btrim($1, %s)`, class), padded).Scan(&got); err != nil {
			t.Fatalf("U+%04X: %v", r, err)
		}
		if want := strings.TrimSpace(padded); got != want {
			t.Errorf("U+%04X: the migration keeps %q, Go gives %q", r, got, want)
		}
	}

	if trimmed == 0 {
		t.Fatal("no whitespace runes were checked")
	}
	t.Logf("checked %d whitespace runes", trimmed)

	// And it must not eat characters that merely look like padding.
	for _, keep := range []string{"aino", "ai no", "-aino-", "aino_"} {
		var got string
		if err := db.QueryRow(fmt.Sprintf(`SELECT btrim($1, %s)`, class), keep).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != keep {
			t.Errorf("over-trimmed: %q became %q", keep, got)
		}
	}
}
