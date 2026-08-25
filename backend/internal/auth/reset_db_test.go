package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

type mailRecorder struct {
	mu   sync.Mutex
	sent []notify.Message
}

func (r *mailRecorder) Notify(_ context.Context, m notify.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, m)
}

func (r *mailRecorder) Close() {}

func (r *mailRecorder) messages() []notify.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]notify.Message(nil), r.sent...)
}

func (r *mailRecorder) resetCount() int {
	n := 0
	for _, m := range r.messages() {
		if m.Kind == notify.KindPasswordReset {
			n++
		}
	}
	return n
}

func newResetService(t *testing.T) (*Service, *database.DB, *mailRecorder) {
	t.Helper()

	db := testdb.New(t)
	rec := &mailRecorder{}
	svc := NewService(db, NewJwtService("test-secret", time.Minute), rec, "http://frontend.test")

	return svc, db, rec
}

// The raw token exists only in the email - the row holds a hash - so reading it
// here also proves it reaches the recipient.
func resetTokenFrom(t *testing.T, m notify.Message) string {
	t.Helper()

	_, after, found := strings.Cut(m.Body, "?token=")
	if !found {
		t.Fatalf("no token in the reset email:\n%s", m.Body)
	}
	token, _, _ := strings.Cut(after, "\n")
	return strings.TrimSpace(token)
}

func requestReset(t *testing.T, svc *Service, rec *mailRecorder, email string) string {
	t.Helper()

	if err := svc.RequestReset(context.Background(), email); err != nil {
		t.Fatalf("requesting a reset: %v", err)
	}
	msgs := rec.messages()
	if len(msgs) == 0 {
		t.Fatal("no reset email was sent")
	}
	return resetTokenFrom(t, msgs[len(msgs)-1])
}

func TestRequestResetLooksTheSameForAnUnknownAddress(t *testing.T) {
	svc, _, rec := newResetService(t)
	ctx := context.Background()

	if _, err := svc.Signup(ctx, signupInput("aino")); err != nil {
		t.Fatalf("signup: %v", err)
	}

	known := svc.RequestReset(ctx, "aino@example.test")
	unknown := svc.RequestReset(ctx, "nobody@example.test")

	if known != nil || unknown != nil {
		t.Errorf("known = %v, unknown = %v - both must be nil", known, unknown)
	}
	if n := rec.resetCount(); n != 1 {
		t.Errorf("sent %d reset emails, want 1", n)
	}
}

func TestResetChangesThePasswordAndRevokesSessions(t *testing.T) {
	svc, _, rec := newResetService(t)
	ctx := context.Background()

	signup, err := svc.Signup(ctx, signupInput("aino"))
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	token := requestReset(t, svc, rec, "aino@example.test")

	if err := svc.ResetPassword(ctx, token, "a-brand-new-password"); err != nil {
		t.Fatalf("resetting: %v", err)
	}

	if _, err := svc.Login(ctx, dtos.LoginRequest{
		Email: "aino@example.test", Password: "password123",
	}); err == nil {
		t.Error("the old password still works")
	}

	if _, err := svc.Login(ctx, dtos.LoginRequest{
		Email: "aino@example.test", Password: "a-brand-new-password",
	}); err != nil {
		t.Errorf("the new password does not work: %v", err)
	}

	if _, err := svc.RedeemSession(ctx, signup.RefreshToken); err == nil {
		t.Error("a session issued before the reset is still redeemable")
	}
}

func TestResetRevokesAPIKeys(t *testing.T) {
	svc, db, rec := newResetService(t)
	ctx := context.Background()

	signup, err := svc.Signup(ctx, signupInput("aino"))
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	userID, err := uuid.Parse(signup.User.ID)
	if err != nil {
		t.Fatalf("parsing the user id: %v", err)
	}

	if _, err := db.CreateKey(ctx, database.CreateKeyParams{
		ID:        database.NewID(),
		UserID:    userID,
		Name:      "attacker",
		KeyPrefix: "fk_test",
		KeyHash:   "hash-of-the-attackers-key",
	}); err != nil {
		t.Fatalf("creating an api key: %v", err)
	}

	if _, err := db.FindLiveKeyByHash(ctx, "hash-of-the-attackers-key"); err != nil {
		t.Fatalf("the key is not live before the reset: %v", err)
	}

	token := requestReset(t, svc, rec, "aino@example.test")
	if err := svc.ResetPassword(ctx, token, "a-brand-new-password"); err != nil {
		t.Fatalf("resetting: %v", err)
	}

	if _, err := db.FindLiveKeyByHash(ctx, "hash-of-the-attackers-key"); err == nil {
		t.Error("an api key created before the reset still authenticates")
	}
}

func TestTheCooldownHoldsUnderConcurrentRequests(t *testing.T) {
	svc, db, rec := newResetService(t)
	ctx := context.Background()

	if _, err := svc.Signup(ctx, signupInput("aino")); err != nil {
		t.Fatalf("signup: %v", err)
	}

	const requests = 40
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_ = svc.RequestReset(ctx, "aino@example.test")
		}()
	}
	close(start)
	wg.Wait()

	if n := rec.resetCount(); n != 1 {
		t.Errorf("sent %d emails for %d simultaneous requests, want 1", n, requests)
	}

	var live int
	if err := db.QueryRow(
		`SELECT count(*) FROM password_reset_tokens WHERE used_at IS NULL AND expires_at > now()`,
	).Scan(&live); err != nil {
		t.Fatalf("counting live tokens: %v", err)
	}
	if live != 1 {
		t.Errorf("%d tokens are live at once, want 1", live)
	}
}

func TestResetRevokesASessionRotatedWithinTheGracePeriod(t *testing.T) {
	svc, _, rec := newResetService(t)
	ctx := context.Background()

	signup, err := svc.Signup(ctx, signupInput("aino"))
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	if _, err := svc.RedeemSession(ctx, signup.RefreshToken); err != nil {
		t.Fatalf("rotating the session: %v", err)
	}
	if _, err := svc.RedeemSession(ctx, signup.RefreshToken); err != nil {
		t.Fatalf("the rotated token should still be inside the grace window: %v", err)
	}

	token := requestReset(t, svc, rec, "aino@example.test")
	if err := svc.ResetPassword(ctx, token, "a-brand-new-password"); err != nil {
		t.Fatalf("resetting: %v", err)
	}

	if _, err := svc.RedeemSession(ctx, signup.RefreshToken); err == nil {
		t.Error("a token rotated within the grace window survived the reset")
	}
}

func TestAResetTokenWorksOnlyOnce(t *testing.T) {
	svc, _, rec := newResetService(t)
	ctx := context.Background()

	if _, err := svc.Signup(ctx, signupInput("aino")); err != nil {
		t.Fatalf("signup: %v", err)
	}
	token := requestReset(t, svc, rec, "aino@example.test")

	if err := svc.ResetPassword(ctx, token, "first-new-password"); err != nil {
		t.Fatalf("first reset: %v", err)
	}

	err := svc.ResetPassword(ctx, token, "second-new-password")

	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("err = %#v, want *AuthError", err)
	}
}

func TestAnExpiredResetTokenIsRefused(t *testing.T) {
	svc, db, rec := newResetService(t)
	ctx := context.Background()

	if _, err := svc.Signup(ctx, signupInput("aino")); err != nil {
		t.Fatalf("signup: %v", err)
	}
	token := requestReset(t, svc, rec, "aino@example.test")

	if _, err := db.Exec(
		`UPDATE password_reset_tokens SET expires_at = now() - interval '1 minute'`,
	); err != nil {
		t.Fatalf("ageing the token: %v", err)
	}

	err := svc.ResetPassword(ctx, token, "a-brand-new-password")

	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("err = %#v, want *AuthError", err)
	}
}

func TestTheCooldownSuppressesASecondRequest(t *testing.T) {
	svc, _, rec := newResetService(t)
	ctx := context.Background()

	if _, err := svc.Signup(ctx, signupInput("aino")); err != nil {
		t.Fatalf("signup: %v", err)
	}

	for range 3 {
		if err := svc.RequestReset(ctx, "aino@example.test"); err != nil {
			t.Fatalf("requesting a reset: %v", err)
		}
	}

	if n := rec.resetCount(); n != 1 {
		t.Errorf("sent %d reset emails for three requests, want 1", n)
	}
}

func TestANewLinkInvalidatesThePreviousOne(t *testing.T) {
	svc, db, rec := newResetService(t)
	ctx := context.Background()

	if _, err := svc.Signup(ctx, signupInput("aino")); err != nil {
		t.Fatalf("signup: %v", err)
	}
	first := requestReset(t, svc, rec, "aino@example.test")

	// Step past the cooldown without sleeping through it.
	if _, err := db.Exec(
		`UPDATE password_reset_tokens SET created_at = now() - interval '1 hour'`,
	); err != nil {
		t.Fatalf("ageing the request: %v", err)
	}

	second := requestReset(t, svc, rec, "aino@example.test")
	if first == second {
		t.Fatal("the second request reused the first token")
	}

	if err := svc.ResetPassword(ctx, first, "a-brand-new-password"); err == nil {
		t.Error("the superseded link still works")
	}
	if err := svc.ResetPassword(ctx, second, "a-brand-new-password"); err != nil {
		t.Errorf("the current link does not work: %v", err)
	}
}

func TestResetRejectsAShortPassword(t *testing.T) {
	svc, _, rec := newResetService(t)
	ctx := context.Background()

	if _, err := svc.Signup(ctx, signupInput("aino")); err != nil {
		t.Fatalf("signup: %v", err)
	}
	token := requestReset(t, svc, rec, "aino@example.test")

	err := svc.ResetPassword(ctx, token, "short")

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("err = %#v, want *ValidationError", err)
	}
}
