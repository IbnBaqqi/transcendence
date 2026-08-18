package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func signUp(t *testing.T) (*Service, *database.DB, dtos.CreateUserRequest, string) {
	t.Helper()

	svc, db := newService(t)
	input := signupInput("aino")

	signup, err := svc.Signup(context.Background(), input)
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	return svc, db, input, signup.RefreshToken
}

func sessionCount(t *testing.T, db *database.DB) int {
	t.Helper()

	var n int
	if err := db.QueryRow(`SELECT count(*) FROM refresh_tokens`).Scan(&n); err != nil {
		t.Fatalf("counting sessions: %v", err)
	}
	return n
}

func TestSignupAndLoginHandOutRedeemableSessions(t *testing.T) {
	svc, _, input, fromSignup := signUp(t)
	ctx := context.Background()

	if _, err := svc.RedeemSession(ctx, fromSignup); err != nil {
		t.Errorf("signup's token was not redeemable: %v", err)
	}

	login, err := svc.Login(ctx, dtos.LoginRequest{Email: input.Email, Password: input.Password})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := svc.RedeemSession(ctx, login.RefreshToken); err != nil {
		t.Errorf("login's token was not redeemable: %v", err)
	}
}

func TestTheRawTokenIsNeverStored(t *testing.T) {
	_, db, _, raw := signUp(t)

	var stored string
	if err := db.QueryRow(`SELECT token_hash FROM refresh_tokens`).Scan(&stored); err != nil {
		t.Fatal(err)
	}

	if stored == raw {
		t.Error("the database holds the token itself")
	}
	if stored == "" {
		t.Error("nothing was stored")
	}
}

func TestRedeemingRotates(t *testing.T) {
	svc, _, _, first := signUp(t)
	ctx := context.Background()

	result, err := svc.RedeemSession(ctx, first)
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}

	if result.RefreshToken == first {
		t.Error("the same refresh token came back - it was not rotated")
	}
	if result.AccessToken == "" {
		t.Error("no access token")
	}
	if result.User.Username != "aino" {
		t.Errorf("user = %q, want aino - the response should identify the session", result.User.Username)
	}

	if _, err := svc.RedeemSession(ctx, result.RefreshToken); err != nil {
		t.Errorf("the rotated-in token was not redeemable: %v", err)
	}
}

func TestARotatedTokenStillWorksInsideTheGraceWindow(t *testing.T) {
	svc, _, _, first := signUp(t)
	ctx := context.Background()

	if _, err := svc.RedeemSession(ctx, first); err != nil {
		t.Fatalf("first redeem: %v", err)
	}

	if _, err := svc.RedeemSession(ctx, first); err != nil {
		t.Errorf("the just-rotated token was refused inside the window: %v", err)
	}
}

func TestARotatedTokenDiesOutsideTheGraceWindow(t *testing.T) {
	svc, db, _, first := signUp(t)
	ctx := context.Background()

	if _, err := svc.RedeemSession(ctx, first); err != nil {
		t.Fatalf("first redeem: %v", err)
	}

	if _, err := db.Exec(`UPDATE refresh_tokens SET revoked_at = now() - interval '1 hour'
	                      WHERE revoked_at IS NOT NULL`); err != nil {
		t.Fatal(err)
	}

	var authErr *AuthError
	if _, err := svc.RedeemSession(ctx, first); !errors.As(err, &authErr) {
		t.Errorf("err = %v, want *AuthError - a replayed token must be refused", err)
	}
}

func TestLogoutEndsTheSessionImmediately(t *testing.T) {
	svc, _, _, raw := signUp(t)
	ctx := context.Background()

	if err := svc.EndSession(ctx, raw); err != nil {
		t.Fatalf("logout: %v", err)
	}

	var authErr *AuthError
	if _, err := svc.RedeemSession(ctx, raw); !errors.As(err, &authErr) {
		t.Errorf("err = %v, want *AuthError - logout should end the session now", err)
	}
}

func TestExpiredUnknownAndEmptyTokensAreRefused(t *testing.T) {
	svc, db, _, raw := signUp(t)
	ctx := context.Background()

	if _, err := db.Exec(`UPDATE refresh_tokens SET expires_at = now() - interval '1 day'`); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{"expired", raw},
		{"unknown", "0123456789abcdef"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var authErr *AuthError
			if _, err := svc.RedeemSession(ctx, tt.token); !errors.As(err, &authErr) {
				t.Errorf("err = %v, want *AuthError", err)
			}
		})
	}
}

func TestARejectedSignupStoresNoSession(t *testing.T) {
	svc, db, input, _ := signUp(t)

	before := sessionCount(t, db)

	if _, err := svc.Signup(context.Background(), input); err == nil {
		t.Fatal("a duplicate signup should have failed")
	}

	if after := sessionCount(t, db); after != before {
		t.Errorf("sessions = %d, want %d", after, before)
	}
}

func TestDeletingAUserEndsTheirSessions(t *testing.T) {
	svc, db, input, _ := signUp(t)
	ctx := context.Background()

	if _, err := svc.Login(ctx, dtos.LoginRequest{Email: input.Email, Password: input.Password}); err != nil {
		t.Fatal(err)
	}
	if sessionCount(t, db) < 2 {
		t.Fatal("expected a session per login")
	}

	if _, err := db.Exec(`DELETE FROM users`); err != nil {
		t.Fatal(err)
	}

	if n := sessionCount(t, db); n != 0 {
		t.Errorf("sessions = %d, want 0", n)
	}
}
