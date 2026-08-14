package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func newService(t *testing.T) (*Service, *database.DB) {
	t.Helper()

	db := testdb.New(t)
	return NewService(db, NewJwtService("test-secret")), db
}

func signupInput(name string) dtos.CreateUserRequest {
	return dtos.CreateUserRequest{
		Username: name,
		Email:    name + "@example.test",
		Password: "password123",
	}
}

func countUsers(t *testing.T, db *database.DB) int {
	t.Helper()

	var n int
	if err := db.QueryRow("SELECT count(*) FROM users").Scan(&n); err != nil {
		t.Fatalf("counting users: %v", err)
	}
	return n
}

func TestSignupCreatesUserAndProfile(t *testing.T) {
	svc, db := newService(t)

	res, err := svc.Signup(context.Background(), signupInput("forager"))
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}
	if res.AccessToken == "" {
		t.Error("no access token issued")
	}

	var hasProfile bool
	err = db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM users u JOIN profiles p ON p.id = u.id WHERE u.email = $1
	)`, "forager@example.test").Scan(&hasProfile)
	if err != nil {
		t.Fatalf("looking up the profile: %v", err)
	}
	if !hasProfile {
		t.Error("user has no profile row - the invariant #87 exists to keep")
	}
}

func TestSignupRollsBackWhenProfileFails(t *testing.T) {
	svc, db := newService(t)

	if _, err := db.Exec("ALTER TABLE profiles RENAME TO profiles_hidden"); err != nil {
		t.Fatalf("hiding the profiles table: %v", err)
	}

	before := countUsers(t, db)

	if _, err := svc.Signup(context.Background(), signupInput("ghost")); err == nil {
		t.Fatalf("signup succeeded despite the profile insert failing")
	}

	if after := countUsers(t, db); after != before {
		t.Errorf("users = %d, want %d - the user survived a failed signup", after, before)
	}
}

func TestSignupRejectsADuplicateEmail(t *testing.T) {
	svc, _ := newService(t)

	if _, err := svc.Signup(context.Background(), signupInput("first")); err != nil {
		t.Fatalf("first signup failed: %v", err)
	}

	second := signupInput("second")
	second.Email = "first@example.test"

	_, err := svc.Signup(context.Background(), second)

	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v, want *ConflictError", err)
	}
}

func TestLoginAcceptsTheSignupPassword(t *testing.T) {
	svc, _ := newService(t)

	input := signupInput("returning")
	if _, err := svc.Signup(context.Background(), input); err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	if _, err := svc.Login(context.Background(), dtos.LoginRequest{
		Email:    input.Email,
		Password: input.Password,
	}); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	_, err := svc.Login(context.Background(), dtos.LoginRequest{
		Email:    input.Email,
		Password: "wrong-password",
	})

	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("err = %v, want *AuthError", err)
	}
}
