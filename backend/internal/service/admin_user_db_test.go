package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
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

func TestASuspendedSellerLeavesTheReadPaths(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()

	listing, err := f.db.CreateListing(ctx, database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: f.member, Title: "Chanterelles", Category: "mushrooms",
		Price: "18.10", Quantity: 10, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating a listing: %v", err)
	}

	listings := NewListingService(f.db, nil)
	profiles := NewProfileService(f.db, nil)

	before, err := listings.SearchListings(ctx, uuid.Nil, dtos.ListingSearchQuery{})
	if err != nil {
		t.Fatalf("browsing: %v", err)
	}
	if len(before.Items) == 0 {
		t.Fatal("the listing is not in browse to begin with, so this proves nothing")
	}

	if _, err := f.admins.Suspend(ctx, f.admin, f.member, "spamming"); err != nil {
		t.Fatalf("suspending: %v", err)
	}

	after, err := listings.SearchListings(ctx, uuid.Nil, dtos.ListingSearchQuery{})
	if err != nil {
		t.Fatalf("browsing: %v", err)
	}
	for _, l := range after.Items {
		if l.ID == listing.ID {
			t.Error("a suspended seller's listing is still in the browse list")
		}
	}

	if _, err := profiles.Get(ctx, f.member); !isNotFound(err) {
		t.Errorf("a suspended user's profile: err = %#v, want *NotFoundError", err)
	}

	if _, err := f.admins.Reinstate(ctx, f.admin, f.member, ""); err != nil {
		t.Fatalf("reinstating: %v", err)
	}

	back, err := listings.SearchListings(ctx, uuid.Nil, dtos.ListingSearchQuery{})
	if err != nil {
		t.Fatalf("browsing: %v", err)
	}
	var found bool
	for _, l := range back.Items {
		if l.ID == listing.ID {
			found = true
		}
	}
	if !found {
		t.Error("reinstating did not bring the listing back")
	}
	if _, err := profiles.Get(ctx, f.member); err != nil {
		t.Errorf("reinstating did not bring the profile back: %v", err)
	}
}

// The guard counts *active* admins, so it must not fire for a subject who is
// already suspended - they are not one of the admins it is protecting, and
// treating them as one blocks the whole point of suspending an admin first.
func TestASuspendedAdminCanStillBeDeleted(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()

	if _, err := f.admins.Suspend(ctx, f.admin, f.other, "abusing the role"); err != nil {
		t.Fatalf("suspending the second admin: %v", err)
	}

	// f.admin is now the only active admin, which is what used to make
	// CountAdmins return 1 and refuse this.
	if err := f.admins.Delete(ctx, f.admin, f.other, "other", "abusing the role"); err != nil {
		t.Fatalf("deleting a suspended admin: %v", err)
	}

	user, err := f.db.GetUser(ctx, f.other)
	if err != nil {
		t.Fatalf("re-reading: %v", err)
	}
	if !user.DeletedAt.Valid {
		t.Error("the account was not deleted")
	}
}

func TestASuspendedUserLeavesTheFollowLists(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()

	if err := f.db.FollowUser(ctx, database.FollowUserParams{
		FollowerID: f.admin, FolloweeID: f.member,
	}); err != nil {
		t.Fatalf("following: %v", err)
	}
	if err := f.db.FollowUser(ctx, database.FollowUserParams{
		FollowerID: f.member, FolloweeID: f.admin,
	}); err != nil {
		t.Fatalf("following back: %v", err)
	}

	if _, err := f.admins.Suspend(ctx, f.admin, f.member, "spamming"); err != nil {
		t.Fatalf("suspending: %v", err)
	}

	following, err := f.db.ListFollowing(ctx, database.ListFollowingParams{
		SubjectID: f.admin, ViewerID: f.admin,
	})
	if err != nil {
		t.Fatalf("listing who the admin follows: %v", err)
	}
	for _, u := range following {
		if u.ID == f.member {
			t.Error("a suspended user is still in someone's following list")
		}
	}

	followers, err := f.db.ListFollowers(ctx, database.ListFollowersParams{
		SubjectID: f.admin, ViewerID: f.admin,
	})
	if err != nil {
		t.Fatalf("listing the admin's followers: %v", err)
	}
	for _, u := range followers {
		if u.ID == f.member {
			t.Error("a suspended user is still in someone's follower list")
		}
	}
}

// The guard reads the roster and then acts on it. GetUserForUpdate locks the
// subject's row only, so what stops two admins suspending each other into an
// empty roster is the advisory lock - and the way to prove the guard takes it
// is to hold it and watch the guard wait.
func TestTheGuardWaitsForTheAdminRosterLock(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()

	holder, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("opening the holding transaction: %v", err)
	}
	defer func() { _ = holder.Rollback() }()

	if err := f.db.Queries.WithTx(holder.Tx).LockAdminRoster(ctx); err != nil {
		t.Fatalf("taking the roster lock: %v", err)
	}

	// Suspending an admin has to wait for the roster, so with the lock held
	// this can only end in the deadline.
	timed, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// The driver reports a cancelled statement as its own error rather than
	// context.DeadlineExceeded, so the deadline itself is what to assert on.
	if _, err := f.admins.Suspend(timed, f.admin, f.other, "concurrent"); err == nil {
		t.Error("suspending an admin did not wait for the roster lock")
	} else if timed.Err() == nil {
		t.Errorf("it failed for some reason other than the wait: %v", err)
	}

	// A plain user is not part of the roster, so that path must not serialise
	// behind admin actions - the lock is taken after the role check.
	if _, err := f.admins.Suspend(ctx, f.admin, f.member, "unrelated"); err != nil {
		t.Errorf("suspending a plain user waited for the roster lock: %v", err)
	}
}
