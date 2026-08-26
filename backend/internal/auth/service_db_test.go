package auth

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func newService(t *testing.T) (*Service, *database.DB) {
	t.Helper()

	db := testdb.New(t)
	return NewService(db, NewJwtService("test-secret", time.Minute)), db
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
	if conflict.Message != "Email already in use" {
		t.Errorf("message = %q, want %q", conflict.Message, "Email already in use")
	}
}

// An account created through an OAuth provider has no password, so no password
// may open it - and the refusal must look exactly like a wrong one. Saying
// "this account uses Google" would confirm the address is registered and tell
// an attacker which door to try.
//
// Note this passes with or without the explicit Valid check in Login: bcrypt
// rejects the empty hash too. The test pins the behaviour a client sees, not
// which line produces it.
func TestLoginRejectsAPasswordlessAccount(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	if _, err := db.Exec(
		`INSERT INTO users (id, email, username) VALUES ($1, $2, $3)`,
		database.NewID(), "oauth@example.test", "oauthonly",
	); err != nil {
		t.Fatal(err)
	}

	// What a wrong password gets, for comparison.
	if _, err := svc.Signup(ctx, signupInput("withpassword")); err != nil {
		t.Fatal(err)
	}
	var wrongPassword *AuthError
	_, err := svc.Login(ctx, dtos.LoginRequest{
		Email: "withpassword@example.test", Password: "not the password",
	})
	if !errors.As(err, &wrongPassword) {
		t.Fatalf("a wrong password gave %v, want *AuthError", err)
	}

	// An empty password is rejected earlier still, as a malformed request, so
	// it never reaches the lookup and cannot leak anything either.
	var invalid *ValidationError
	if _, err := svc.Login(ctx, dtos.LoginRequest{Email: "oauth@example.test"}); !errors.As(err, &invalid) {
		t.Errorf("an empty password gave %v, want *ValidationError", err)
	}

	for _, attempt := range []string{"password123", "null", "NULL", "seed-placeholder-password"} {
		t.Run("password "+attempt, func(t *testing.T) {
			_, err := svc.Login(ctx, dtos.LoginRequest{
				Email: "oauth@example.test", Password: attempt,
			})

			var authErr *AuthError
			if !errors.As(err, &authErr) {
				t.Fatalf("err = %v, want *AuthError", err)
			}
			if authErr.Message != wrongPassword.Message {
				t.Errorf("message = %q, want %q - it must not reveal that the\n"+
					"account exists but has no password", authErr.Message, wrongPassword.Message)
			}
		})
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

func TestSignupRejectsADuplicateUsername(t *testing.T) {
	svc, db := newService(t)

	if _, err := svc.Signup(context.Background(), signupInput("taken")); err != nil {
		t.Fatalf("first signup failed: %v", err)
	}

	before := countUsers(t, db)

	second := signupInput("taken")
	second.Email = "someone.else@example.test"

	_, err := svc.Signup(context.Background(), second)

	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v, want *ConflictError", err)
	}
	if conflict.Message != "Username already taken" {
		t.Errorf("message = %q, want %q", conflict.Message, "Username already taken")
	}
	if after := countUsers(t, db); after != before {
		t.Errorf("users = %d, want %d - the rejected signup left a row behind", after, before)
	}
}

func TestSignupRejectsACaseVariantUsername(t *testing.T) {
	svc, db := newService(t)

	first := signupInput("forager")
	first.Username = "Forager"
	if _, err := svc.Signup(context.Background(), first); err != nil {
		t.Fatalf("first signup failed: %v", err)
	}

	before := countUsers(t, db)

	second := signupInput("forager")
	second.Username = "forager"
	second.Email = "someone.else@example.test"

	_, err := svc.Signup(context.Background(), second)

	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v, want *ConflictError", err)
	}
	if conflict.Message != "Username already taken" {
		t.Errorf("message = %q, want %q", conflict.Message, "Username already taken")
	}
	if after := countUsers(t, db); after != before {
		t.Errorf("users = %d, want %d", after, before)
	}
}

func TestSignupRejectsACaseVariantEmail(t *testing.T) {
	svc, _ := newService(t)

	first := signupInput("aino")
	first.Email = "Aino@Example.test"
	if _, err := svc.Signup(context.Background(), first); err != nil {
		t.Fatalf("first signup failed: %v", err)
	}

	second := signupInput("someoneelse")
	second.Email = "aino@example.test"

	_, err := svc.Signup(context.Background(), second)

	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v, want *ConflictError", err)
	}
	if conflict.Message != "Email already in use" {
		t.Errorf("message = %q, want %q", conflict.Message, "Email already in use")
	}
}

func TestTheUniqueIndexIsTheBackstop(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	first := signupInput("forager")
	first.Username = "Forager"
	if _, err := svc.Signup(ctx, first); err != nil {
		t.Fatalf("first signup failed: %v", err)
	}

	_, err := db.Queries.CreateUser(ctx, database.CreateUserParams{
		ID:       database.NewID(),
		Username: "forager",
		Email:    "someone.else@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err == nil {
		t.Fatal("the index let a case-variant username through")
	}

	var conflict *ConflictError
	if !errors.As(duplicateUserError(err), &conflict) {
		t.Fatalf("err = %v, want it mapped to *ConflictError - unmapped is a 500", err)
	}
}

func TestTheEmailIndexIsTheBackstop(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	first := signupInput("aino")
	first.Email = "Aino@Example.test"
	if _, err := svc.Signup(ctx, first); err != nil {
		t.Fatalf("first signup failed: %v", err)
	}

	_, err := db.Queries.CreateUser(ctx, database.CreateUserParams{
		ID:       database.NewID(),
		Username: "someoneelse",
		Email:    "aino@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err == nil {
		t.Fatal("the index let a case-variant email through")
	}

	var conflict *ConflictError
	if !errors.As(duplicateUserError(err), &conflict) {
		t.Fatalf("err = %v, want it mapped to *ConflictError - unmapped is a 500", err)
	}
	if conflict.Message != "Email already in use" {
		t.Errorf("message = %q, want %q", conflict.Message, "Email already in use")
	}
}

func TestLoginAcceptsADifferentlyCasedEmail(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	in := signupInput("aino")
	in.Email = "Aino@Example.test"
	if _, err := svc.Signup(ctx, in); err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	if _, err := svc.Login(ctx, dtos.LoginRequest{
		Email:    "  aino@example.TEST  ",
		Password: in.Password,
	}); err != nil {
		t.Errorf("login failed: %v", err)
	}
}

func TestSignupStoresTheTrimmedValueWithCaseIntact(t *testing.T) {
	svc, db := newService(t)

	in := signupInput("ignored")
	in.Username = "  Aino  "
	in.Email = "  Aino@Example.test  "
	res, err := svc.Signup(context.Background(), in)
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	var name, email string
	if err := db.QueryRow(`SELECT username, email FROM users WHERE id = $1`, res.User.ID).Scan(&name, &email); err != nil {
		t.Fatal(err)
	}
	if name != "Aino" {
		t.Errorf("username = %q, want %q", name, "Aino")
	}
	if email != "Aino@Example.test" {
		t.Errorf("email = %q, want %q", email, "Aino@Example.test")
	}
}

func TestSignupRejectsADuplicateOfBothFields(t *testing.T) {
	svc, db := newService(t)

	input := signupInput("clash")
	if _, err := svc.Signup(context.Background(), input); err != nil {
		t.Fatalf("first signup failed: %v", err)
	}

	before := countUsers(t, db)

	_, err := svc.Signup(context.Background(), input)

	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v, want *ConflictError", err)
	}
	if conflict.Message != "Email already in use" && conflict.Message != "Username already taken" {
		t.Errorf("message = %q, want one of the two duplicate messages", conflict.Message)
	}
	if after := countUsers(t, db); after != before {
		t.Errorf("users = %d, want %d - the rejected signup left a row behind", after, before)
	}
}
