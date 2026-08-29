package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

type adminFixture struct {
	admins *AdminUserService
	db     *database.DB
	admin  uuid.UUID
	other  uuid.UUID
	member uuid.UUID
}

func newAdminFixture(t *testing.T) adminFixture {
	t.Helper()

	db := testdb.New(t)
	ctx := context.Background()

	mk := func(name, role string) uuid.UUID {
		user, err := db.CreateUser(ctx, database.CreateUserParams{
			ID:       database.NewID(),
			Username: name, Email: name + "@example.test",
			Password: sql.NullString{String: "irrelevant", Valid: true},
		})
		if err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		if err := db.EnsureProfile(ctx, user.ID); err != nil {
			t.Fatalf("creating %s's profile: %v", name, err)
		}
		if role == "ADMIN" {
			if _, err := db.ExecContext(ctx, `UPDATE users SET role = 'ADMIN' WHERE id = $1`, user.ID); err != nil {
				t.Fatalf("promoting %s: %v", name, err)
			}
		}
		return user.ID
	}

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	return adminFixture{
		admins: NewAdminUserService(db, files),
		db:     db,
		admin:  mk("admin", "ADMIN"),
		other:  mk("other", "ADMIN"),
		member: mk("member", "USER"),
	}
}

func TestSuspensionIsReversibleAndKeepsEverything(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()

	before, err := f.db.GetUser(ctx, f.member)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	suspended, err := f.admins.Suspend(ctx, f.admin, f.member, "spamming listings")
	if err != nil {
		t.Fatalf("suspending: %v", err)
	}
	if !suspended.SuspendedAt.Valid {
		t.Error("suspended_at was not set")
	}
	if suspended.SuspensionReason.String != "spamming listings" {
		t.Errorf("reason = %q, want it stored for the appeal", suspended.SuspensionReason.String)
	}

	active, err := f.db.UserIsActive(ctx, f.member)
	if err != nil {
		t.Fatalf("checking active: %v", err)
	}
	if active {
		t.Error("a suspended account still reads as active, so nothing enforces it")
	}

	// Nothing was scrubbed.
	if suspended.Email != before.Email || suspended.Username != before.Username {
		t.Error("suspension changed identifying fields - it is meant to be lossless")
	}

	back, err := f.admins.Reinstate(ctx, f.admin, f.member, "appealed successfully")
	if err != nil {
		t.Fatalf("reinstating: %v", err)
	}
	if back.SuspendedAt.Valid || back.SuspensionReason.Valid {
		t.Error("reinstating did not clear the suspension")
	}
	if back.Email != before.Email {
		t.Error("the account did not come back whole")
	}
}

func TestAnAdminCannotActOnThemselves(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()

	var forbidden *ForbiddenError

	if _, err := f.admins.Suspend(ctx, f.admin, f.admin, "oops"); !errors.As(err, &forbidden) {
		t.Errorf("self-suspend: err = %#v, want *ForbiddenError", err)
	}
	if err := f.admins.Delete(ctx, f.admin, f.admin, "admin", "oops"); !errors.As(err, &forbidden) {
		t.Errorf("self-delete: err = %#v, want *ForbiddenError", err)
	}
}

func TestTheLastAdminCannotBeRemoved(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()

	if _, err := f.admins.Suspend(ctx, f.admin, f.other, "under review"); err != nil {
		t.Fatalf("suspending the second admin: %v", err)
	}

	if _, err := f.db.ExecContext(ctx, `UPDATE users SET role = 'ADMIN' WHERE id = $1`, f.member); err != nil {
		t.Fatalf("promoting the member: %v", err)
	}

	if _, err := f.admins.Suspend(ctx, f.member, f.admin, "trying"); err != nil {
		t.Fatalf("suspending with two admins active should work: %v", err)
	}

	var conflict *ConflictError
	if _, err := f.admins.Suspend(ctx, f.admin, f.member, "last one"); !errors.As(err, &conflict) {
		t.Errorf("suspending the last admin: err = %#v, want *ConflictError", err)
	}
}

func TestSuspendingTwiceIsAConflict(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()

	if _, err := f.admins.Suspend(ctx, f.admin, f.member, "first"); err != nil {
		t.Fatalf("suspending: %v", err)
	}

	var conflict *ConflictError
	if _, err := f.admins.Suspend(ctx, f.admin, f.member, "second"); !errors.As(err, &conflict) {
		t.Errorf("err = %#v, want *ConflictError", err)
	}

	user, err := f.db.GetUser(ctx, f.member)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if user.SuspensionReason.String != "first" {
		t.Errorf("reason = %q, want the original", user.SuspensionReason.String)
	}

	actions, err := f.admins.History(ctx, f.member)
	if err != nil {
		t.Fatalf("reading history: %v", err)
	}
	if len(actions) != 1 {
		t.Errorf("audit rows = %d, want 1", len(actions))
	}
}

func TestASuspensionNeedsAReason(t *testing.T) {
	f := newAdminFixture(t)

	var invalid *ValidationError
	if _, err := f.admins.Suspend(context.Background(), f.admin, f.member, "  "); !errors.As(err, &invalid) {
		t.Errorf("err = %#v, want *ValidationError - the reason is what an appeal answers", err)
	}
}

func TestAdminDeletionScrubsAndAudits(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()

	if err := f.admins.Delete(ctx, f.admin, f.member, "member", "repeated abuse"); err != nil {
		t.Fatalf("deleting: %v", err)
	}

	user, err := f.db.GetUser(ctx, f.member)
	if err != nil {
		t.Fatalf("the row did not survive: %v", err)
	}
	if !user.DeletedAt.Valid || user.Password.Valid || user.Username == "member" {
		t.Errorf("an admin deletion did not scrub like a self-deletion: %+v", user)
	}

	actions, err := f.admins.History(ctx, f.member)
	if err != nil {
		t.Fatalf("reading history: %v", err)
	}
	if len(actions) != 1 || actions[0].Action != "deleted" {
		t.Fatalf("audit = %+v, want one 'deleted' row", actions)
	}
	if actions[0].Note.String != "repeated abuse" {
		t.Errorf("note = %q, want the reason", actions[0].Note.String)
	}
	if !actions[0].ModeratorID.Valid || actions[0].ModeratorID.UUID != f.admin {
		t.Error("the audit row does not name the admin who acted")
	}
}

func TestAdminDeletionNeedsTheUsernameTyped(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()

	var invalid *ValidationError
	for _, wrong := range []string{"", "Member", "member ", "admin"} {
		if err := f.admins.Delete(ctx, f.admin, f.member, wrong, "reason"); !errors.As(err, &invalid) {
			t.Errorf("confirmation %q: err = %#v, want *ValidationError", wrong, err)
		}
	}

	user, err := f.db.GetUser(ctx, f.member)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if user.DeletedAt.Valid {
		t.Error("a refused deletion still deleted the account")
	}
}
