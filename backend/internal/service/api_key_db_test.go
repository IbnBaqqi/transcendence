package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func newKeyService(t *testing.T) (*APIKeyService, *database.DB, uuid.UUID, uuid.UUID) {
	t.Helper()

	db := testdb.New(t)
	mk := func(name string) uuid.UUID {
		id := uuid.New()
		if _, err := db.Exec(
			`INSERT INTO users (id, email, username, password) VALUES ($1, $2, $3, 'x')`,
			id, name+"@example.test", name,
		); err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		return id
	}

	return NewAPIKeyService(db.Queries), db, mk("aino"), mk("veikko")
}

func TestCreateStoresOnlyAHash(t *testing.T) {
	svc, db, aino, _ := newKeyService(t)

	issued, err := svc.Create(context.Background(), aino, "ci pipeline")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(issued.Key, KeyPrefix) {
		t.Errorf("key = %q, want the %q prefix", issued.Key, KeyPrefix)
	}

	var stored, prefix string
	if err := db.QueryRow(`SELECT key_hash, key_prefix FROM api_keys WHERE id = $1`, issued.Record.ID).
		Scan(&stored, &prefix); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256([]byte(issued.Key))
	if stored != hex.EncodeToString(sum[:]) {
		t.Errorf("stored value is not the key's SHA-256")
	}
	if strings.Contains(stored, issued.Key) {
		t.Error("the raw key is recoverable from what was stored")
	}
	if want := issued.Key[:prefixLength]; prefix != want {
		t.Errorf("key_prefix = %q, want %q", prefix, want)
	}
}

func TestCreateRejectsABlankName(t *testing.T) {
	svc, _, aino, _ := newKeyService(t)

	_, err := svc.Create(context.Background(), aino, "   ")

	var invalid *ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("err = %v, want *ValidationError", err)
	}
}

func TestAuthenticateAcceptsAFreshKey(t *testing.T) {
	svc, _, aino, _ := newKeyService(t)
	ctx := context.Background()

	issued, err := svc.Create(ctx, aino, "ci")
	if err != nil {
		t.Fatal(err)
	}

	id, user, err := svc.Authenticate(ctx, issued.Key)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if id != issued.Record.ID {
		t.Errorf("key id = %d, want %d", id, issued.Record.ID)
	}
	if user.ID != aino {
		t.Errorf("user = %v, want %v", user.ID, aino)
	}
	if user.Name != "aino" || user.Role != "USER" {
		t.Errorf("user = %+v, want aino/USER", user)
	}
}

func TestAuthenticateRejects(t *testing.T) {
	svc, _, aino, _ := newKeyService(t)
	ctx := context.Background()

	revoked, err := svc.Create(ctx, aino, "old")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Revoke(ctx, aino, revoked.Record.ID); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"garbage", "hunter2"},
		{"right shape, wrong prefix", "sk_live_" + strings.Repeat("a", 64)},
		{"unknown", KeyPrefix + strings.Repeat("a", 64)},
		{"revoked", revoked.Key},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := svc.Authenticate(ctx, tt.key)
			if !errors.Is(err, ErrKeyNotUsable) {
				t.Errorf("err = %v, want ErrKeyNotUsable", err)
			}
		})
	}
}

func TestRevocationTakesEffectImmediately(t *testing.T) {
	svc, _, aino, _ := newKeyService(t)
	ctx := context.Background()

	issued, err := svc.Create(ctx, aino, "ci")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Authenticate(ctx, issued.Key); err != nil {
		t.Fatalf("before revocation: %v", err)
	}

	if err := svc.Revoke(ctx, aino, issued.Record.ID); err != nil {
		t.Fatal(err)
	}

	if _, _, err := svc.Authenticate(ctx, issued.Key); !errors.Is(err, ErrKeyNotUsable) {
		t.Errorf("after revocation: err = %v, want ErrKeyNotUsable", err)
	}
}

func TestRevokeIsOwnerScoped(t *testing.T) {
	svc, db, aino, veikko := newKeyService(t)
	ctx := context.Background()

	issued, err := svc.Create(ctx, aino, "ci")
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Revoke(ctx, veikko, issued.Record.ID)

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want *NotFoundError", err)
	}

	var revokedAt any
	if err := db.QueryRow(`SELECT revoked_at FROM api_keys WHERE id = $1`, issued.Record.ID).Scan(&revokedAt); err != nil {
		t.Fatal(err)
	}
	if revokedAt != nil {
		t.Error("veikko revoked a key that is not his")
	}
}

func TestRevokeTwiceIsNotFound(t *testing.T) {
	svc, _, aino, _ := newKeyService(t)
	ctx := context.Background()

	issued, err := svc.Create(ctx, aino, "ci")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Revoke(ctx, aino, issued.Record.ID); err != nil {
		t.Fatal(err)
	}

	var notFound *NotFoundError
	if err := svc.Revoke(ctx, aino, issued.Record.ID); !errors.As(err, &notFound) {
		t.Errorf("err = %v, want *NotFoundError", err)
	}
}

func TestListShowsRevokedKeysAndNoSecrets(t *testing.T) {
	svc, _, aino, _ := newKeyService(t)
	ctx := context.Background()

	live, err := svc.Create(ctx, aino, "live")
	if err != nil {
		t.Fatal(err)
	}
	dead, err := svc.Create(ctx, aino, "dead")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Revoke(ctx, aino, dead.Record.ID); err != nil {
		t.Fatal(err)
	}

	rows, err := svc.List(ctx, aino)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("listed %d keys, want 2 - a revoked key is still worth showing", len(rows))
	}
	for _, row := range rows {
		if row.KeyPrefix == "" {
			t.Error("no prefix, so the user cannot tell which key this is")
		}
		if strings.Contains(row.KeyPrefix, live.Key[prefixLength:]) {
			t.Error("the prefix carries more than the prefix")
		}
	}
}
