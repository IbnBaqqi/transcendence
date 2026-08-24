package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/oauth"
)

func googleLogin(providerUserID, email string) OAuthLogin {
	return OAuthLogin{
		Provider:       oauth.ProviderGoogle,
		ProviderUserID: providerUserID,
		Email:          email,
	}
}

func TestOAuthFirstSignInCreatesUserProfileAndIdentity(t *testing.T) {
	svc, db := newService(t)

	res, err := svc.LoginWithIdentity(context.Background(), googleLogin("g-1", "aino@example.test"))
	if err != nil {
		t.Fatalf("oauth sign-in failed: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Error("no session issued")
	}
	if res.User.Email != "aino@example.test" {
		t.Errorf("user email = %q, want aino@example.test", res.User.Email)
	}
	if res.User.Username != "aino" {
		t.Errorf("username = %q, want it derived from the email", res.User.Username)
	}

	var hasProfile bool
	if err := db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM users u JOIN profiles p ON p.id = u.id WHERE u.email = $1
	)`, "aino@example.test").Scan(&hasProfile); err != nil {
		t.Fatalf("looking up the profile: %v", err)
	}
	if !hasProfile {
		t.Error("oauth user has no profile row")
	}

	var passwordIsNull bool
	if err := db.QueryRow(
		`SELECT password IS NULL FROM users WHERE email = $1`, "aino@example.test",
	).Scan(&passwordIsNull); err != nil {
		t.Fatalf("reading the password column: %v", err)
	}
	if !passwordIsNull {
		t.Error("oauth account was given a password - Login's password-less check depends on it staying NULL")
	}

	var linked bool
	if err := db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM oauth_identities WHERE provider = $1 AND provider_user_id = $2
	)`, oauth.ProviderGoogle, "g-1").Scan(&linked); err != nil {
		t.Fatalf("looking up the identity: %v", err)
	}
	if !linked {
		t.Error("no identity row was written")
	}
}

func TestOAuthReturningUserReusesTheSameAccount(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	first, err := svc.LoginWithIdentity(ctx, googleLogin("g-1", "aino@example.test"))
	if err != nil {
		t.Fatalf("first sign-in failed: %v", err)
	}

	second, err := svc.LoginWithIdentity(ctx, googleLogin("g-1", "aino-new@example.test"))
	if err != nil {
		t.Fatalf("second sign-in failed: %v", err)
	}

	if first.User.ID != second.User.ID {
		t.Errorf("second sign-in produced a different user: %s then %s", first.User.ID, second.User.ID)
	}
	if n := countUsers(t, db); n != 1 {
		t.Errorf("user count = %d, want 1", n)
	}
}

func TestOAuthLinksToAPasswordlessAccountWithTheSameEmail(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	if _, err := svc.LoginWithIdentity(ctx, googleLogin("g-1", "aino@example.test")); err != nil {
		t.Fatalf("google sign-in failed: %v", err)
	}

	github := OAuthLogin{
		Provider:       oauth.ProviderGitHub,
		ProviderUserID: "gh-1",
		Email:          "aino@example.test",
	}
	res, err := svc.LoginWithIdentity(ctx, github)
	if err != nil {
		t.Fatalf("github sign-in failed: %v", err)
	}

	if n := countUsers(t, db); n != 1 {
		t.Errorf("user count = %d, want the second provider linked to the existing account", n)
	}

	var identities int
	if err := db.QueryRow(
		`SELECT count(*) FROM oauth_identities WHERE user_id = $1`, res.User.ID,
	).Scan(&identities); err != nil {
		t.Fatalf("counting identities: %v", err)
	}
	if identities != 2 {
		t.Errorf("identity count = %d, want both providers linked", identities)
	}
}

// The account pre-hijacking case.
func TestOAuthRefusesToLinkAnAccountThatHasAPassword(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	input := signupInput("victim")
	if _, err := svc.Signup(ctx, input); err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	_, err := svc.LoginWithIdentity(ctx, googleLogin("g-1", input.Email))

	var exists *AccountExistsError
	if !errors.As(err, &exists) {
		t.Fatalf("error = %v, want an AccountExistsError", err)
	}

	var identities int
	if err := db.QueryRow(`SELECT count(*) FROM oauth_identities`).Scan(&identities); err != nil {
		t.Fatalf("counting identities: %v", err)
	}
	if identities != 0 {
		t.Error("an identity was linked to a password account")
	}
	if n := countUsers(t, db); n != 1 {
		t.Errorf("user count = %d, want the signup account untouched", n)
	}
}

func TestOAuthRefusesAnEmptyEmail(t *testing.T) {
	svc, db := newService(t)

	_, err := svc.LoginWithIdentity(context.Background(), googleLogin("g-1", ""))

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want a ValidationError", err)
	}
	if n := countUsers(t, db); n != 0 {
		t.Errorf("user count = %d, want no account created without an address", n)
	}
}

func TestOAuthWorksAroundATakenUsername(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	if _, err := svc.Signup(ctx, signupInput("aino")); err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	res, err := svc.LoginWithIdentity(ctx, googleLogin("g-1", "aino@other.test"))
	if err != nil {
		t.Fatalf("oauth sign-in failed: %v", err)
	}

	if res.User.Username == "aino" {
		t.Error("the taken username was reused")
	}
	if n := countUsers(t, db); n != 2 {
		t.Errorf("user count = %d, want 2", n)
	}
}

func TestOAuthRefusesASecondAccountFromTheSameProvider(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	res, err := svc.LoginWithIdentity(ctx, googleLogin("g-1", "aino@example.test"))
	if err != nil {
		t.Fatalf("first sign-in failed: %v", err)
	}

	userID, err := uuid.Parse(res.User.ID)
	if err != nil {
		t.Fatalf("parsing the user id: %v", err)
	}

	if err := db.LinkIdentity(ctx, database.LinkIdentityParams{
		Provider:       oauth.ProviderGoogle,
		ProviderUserID: "g-2",
		UserID:         userID,
	}); err == nil {
		t.Fatal("linking a second google account to the same user was allowed")
	}
}
