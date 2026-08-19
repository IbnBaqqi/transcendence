package database_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

const checkViolation = "23514"

func TestOnlyKnownRolesAreStorable(t *testing.T) {
	db := testdb.New(t)

	id := uuid.New()
	if _, err := db.Exec(
		`INSERT INTO users (id, email, username, password) VALUES ($1, 'aino@example.test', 'aino', 'x')`,
		id,
	); err != nil {
		t.Fatal(err)
	}

	var role string
	if err := db.QueryRow(`SELECT role FROM users WHERE id = $1`, id).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if role != "USER" {
		t.Errorf("default role = %q, want %q", role, "USER")
	}

	if _, err := db.Exec(`UPDATE users SET role = 'ADMIN' WHERE id = $1`, id); err != nil {
		t.Fatalf("promoting to ADMIN should be allowed: %v", err)
	}

	for _, bad := range []string{"admn", "admin", "ADMIN ", "", "SUPERUSER"} {
		t.Run("rejects "+bad, func(t *testing.T) {
			_, err := db.Exec(`UPDATE users SET role = $2 WHERE id = $1`, id, bad)

			var pqErr *pq.Error
			if !errors.As(err, &pqErr) {
				t.Fatalf("err = %v, want a *pq.Error rejecting %q", err, bad)
			}
			if got := string(pqErr.Code); got != checkViolation {
				t.Errorf("SQLSTATE = %s, want %s", got, checkViolation)
			}
			if pqErr.Constraint != "users_role_check" {
				t.Errorf("constraint = %q, want %q", pqErr.Constraint, "users_role_check")
			}
		})
	}
}
