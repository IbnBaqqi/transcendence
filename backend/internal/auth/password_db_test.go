package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func signedUpUser(t *testing.T, svc *Service) (SignupResponse, dtos.LoginRequest) {
	t.Helper()

	in := signupInput("aino")
	res, err := svc.Signup(context.Background(), in)
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	return res, dtos.LoginRequest{Email: in.Email, Password: in.Password}
}

func TestChangingThePasswordSwapsWhichOneWorks(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	_, login := signedUpUser(t, svc)

	user, err := db.GetUserByEmail(ctx, login.Email)
	if err != nil {
		t.Fatalf("looking up the user: %v", err)
	}

	if _, err := svc.ChangePassword(ctx, user.ID, login.Password, "a-brand-new-password"); err != nil {
		t.Fatalf("changing the password: %v", err)
	}

	if _, err := svc.Login(ctx, login); err == nil {
		t.Error("the old password still signs in")
	}

	if _, err := svc.Login(ctx, dtos.LoginRequest{
		Email: login.Email, Password: "a-brand-new-password",
	}); err != nil {
		t.Errorf("the new password does not sign in: %v", err)
	}
}

func TestTheWrongCurrentPasswordChangesNothing(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	_, login := signedUpUser(t, svc)

	user, err := db.GetUserByEmail(ctx, login.Email)
	if err != nil {
		t.Fatalf("looking up the user: %v", err)
	}

	_, err = svc.ChangePassword(ctx, user.ID, "not-the-current-one", "a-brand-new-password")

	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("error = %v, want an auth error", err)
	}

	// The refusal has to happen before the write, not merely be reported.
	if _, err := svc.Login(ctx, login); err != nil {
		t.Errorf("the original password stopped working after a refused change: %v", err)
	}
}

func TestAProviderAccountHasNoPasswordToChange(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	if _, err := svc.LoginWithIdentity(ctx, googleLogin("g-1", "aino@example.test")); err != nil {
		t.Fatalf("oauth sign-in: %v", err)
	}

	user, err := db.GetUserByEmail(ctx, "aino@example.test")
	if err != nil {
		t.Fatalf("looking up the user: %v", err)
	}

	_, err = svc.ChangePassword(ctx, user.ID, "anything", "a-brand-new-password")

	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("error = %v, want a conflict", err)
	}

	var passwordIsNull bool
	if err := db.QueryRow(
		`SELECT password IS NULL FROM users WHERE id = $1`, user.ID,
	).Scan(&passwordIsNull); err != nil {
		t.Fatalf("reading the password column: %v", err)
	}
	if !passwordIsNull {
		t.Error("the provider account was given a password")
	}
}

func TestChangingThePasswordEndsEveryOtherSession(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	signup, login := signedUpUser(t, svc)

	second, err := svc.Login(ctx, login)
	if err != nil {
		t.Fatalf("second sign-in: %v", err)
	}

	user, err := db.GetUserByEmail(ctx, login.Email)
	if err != nil {
		t.Fatalf("looking up the user: %v", err)
	}

	fresh, err := svc.ChangePassword(ctx, user.ID, login.Password, "a-brand-new-password")
	if err != nil {
		t.Fatalf("changing the password: %v", err)
	}

	if _, err := svc.RedeemSession(ctx, signup.RefreshToken); err == nil {
		t.Error("the session that existed before the change still refreshes")
	}
	if _, err := svc.RedeemSession(ctx, second.RefreshToken); err == nil {
		t.Error("the other device's session still refreshes")
	}

	// Issued after the revocation, so it survives it. The other order revokes
	// the new session too, and the request logs out its own caller.
	if _, err := svc.RedeemSession(ctx, fresh); err != nil {
		t.Errorf("the session handed back by the change does not work: %v", err)
	}
}

func TestChangingThePasswordRevokesApiKeys(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	_, login := signedUpUser(t, svc)

	user, err := db.GetUserByEmail(ctx, login.Email)
	if err != nil {
		t.Fatalf("looking up the user: %v", err)
	}

	if _, err := db.CreateKey(ctx, database.CreateKeyParams{
		ID:        database.NewID(),
		UserID:    user.ID,
		Name:      "a key",
		KeyHash:   "hash-of-a-key",
		KeyPrefix: "fgk_abc",
	}); err != nil {
		t.Fatalf("creating an api key: %v", err)
	}

	if _, err := svc.ChangePassword(ctx, user.ID, login.Password, "a-brand-new-password"); err != nil {
		t.Fatalf("changing the password: %v", err)
	}

	var live int
	if err := db.QueryRow(
		`SELECT count(*) FROM api_keys WHERE user_id = $1 AND revoked_at IS NULL`, user.ID,
	).Scan(&live); err != nil {
		t.Fatalf("counting keys: %v", err)
	}
	if live != 0 {
		t.Errorf("live api keys = %d, want none - a key outliving the password it guarded is the point", live)
	}
}

func TestARejectedNewPasswordLeavesTheOldOneWorking(t *testing.T) {
	svc, db := newService(t)
	ctx := context.Background()

	_, login := signedUpUser(t, svc)

	user, err := db.GetUserByEmail(ctx, login.Email)
	if err != nil {
		t.Fatalf("looking up the user: %v", err)
	}

	_, err = svc.ChangePassword(ctx, user.ID, login.Password, "short")

	var invalid *ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("error = %v, want a validation error", err)
	}

	if _, err := svc.Login(ctx, login); err != nil {
		t.Errorf("the original password stopped working after a rejected change: %v", err)
	}
}
