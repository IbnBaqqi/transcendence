package service

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

type moderationFixture struct {
	mod      *ModerationService
	reports  *ReportService
	orders   *OrderService
	chat     *ConversationService
	saved    *SavedListingService
	listings *ListingService
	db       *database.DB
	files    *storage.Local
	admin    uuid.UUID
	seller   uuid.UUID
	buyer    uuid.UUID
	other    uuid.UUID
	listing  uuid.UUID
}

func newModerationFixture(t *testing.T) moderationFixture {
	t.Helper()

	db := testdb.New(t)
	ctx := context.Background()

	mk := func(name string) uuid.UUID {
		user, err := db.CreateUser(ctx, database.CreateUserParams{
			ID:       database.NewID(),
			Username: name, Email: name + "@example.test",
			Password: sql.NullString{String: "irrelevant", Valid: true},
		})
		if err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		return user.ID
	}
	admin, seller, buyer, other := mk("admin"), mk("seller"), mk("buyer"), mk("other")

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: seller, Title: "Chanterelles", Category: "mushrooms",
		Price: "18.10", Quantity: 5, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating a listing: %v", err)
	}

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	return moderationFixture{
		mod:      NewModerationService(db, files),
		reports:  NewReportService(db.Queries),
		orders:   NewOrderService(db, notify.Disabled{}),
		chat:     NewConversationService(db, notify.Disabled{}),
		saved:    NewSavedListingService(db.Queries),
		listings: NewListingService(db, files),
		db:       db,
		files:    files,
		admin:    admin, seller: seller, buyer: buyer, other: other,
		listing: listing.ID,
	}
}

func (f moderationFixture) reportBy(t *testing.T, reporter uuid.UUID) {
	t.Helper()
	if err := f.reports.Report(context.Background(), reporter, f.listing, "prohibited", ""); err != nil {
		t.Fatalf("reporting: %v", err)
	}
}

func (f moderationFixture) remove(t *testing.T) (database.Listing, int64) {
	t.Helper()
	l, n, err := f.mod.Moderate(context.Background(), f.admin, f.listing, "remove", "prohibited species")
	if err != nil {
		t.Fatalf("removing: %v", err)
	}
	return l, n
}

func isNotFound(err error) bool {
	var notFound *NotFoundError
	return errors.As(err, &notFound)
}

func TestOneRemovalResolvesEveryOpenReport(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	f.reportBy(t, f.buyer)
	f.reportBy(t, f.other)

	listing, resolved := f.remove(t)

	if !listing.RemovedAt.Valid {
		t.Error("the listing is not marked removed")
	}
	if resolved != 2 {
		t.Errorf("resolved = %d, want 2 - one action settles every report on the listing", resolved)
	}

	rows, err := f.db.ListReportsForListing(ctx, f.listing)
	if err != nil {
		t.Fatalf("listing reports: %v", err)
	}
	for _, r := range rows {
		if r.Status != "upheld" {
			t.Errorf("report %s = %q, want upheld", r.ID, r.Status)
		}
	}
}

func TestTheQueueGroupsByListingNotByReport(t *testing.T) {
	f := newModerationFixture(t)

	f.reportBy(t, f.buyer)
	f.reportBy(t, f.other)

	queue, err := f.mod.Queue(context.Background())
	if err != nil {
		t.Fatalf("reading the queue: %v", err)
	}

	if len(queue) != 1 {
		t.Fatalf("queue = %d rows, want 1 - two reports on one listing is one problem", len(queue))
	}
	if queue[0].ReportCount != 2 {
		t.Errorf("report_count = %d, want 2", queue[0].ReportCount)
	}
	if queue[0].ListingID != f.listing {
		t.Errorf("queue row is for the wrong listing")
	}
}

func TestResolvedListingsLeaveTheQueue(t *testing.T) {
	f := newModerationFixture(t)

	f.reportBy(t, f.buyer)
	f.remove(t)

	queue, err := f.mod.Queue(context.Background())
	if err != nil {
		t.Fatalf("reading the queue: %v", err)
	}
	if len(queue) != 0 {
		t.Errorf("queue still has %d rows after the decision", len(queue))
	}
}

func TestTheAuditRowRecordsWhoAndWhat(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	f.reportBy(t, f.buyer)
	f.remove(t)

	actions, err := f.mod.History(ctx, f.listing)
	if err != nil {
		t.Fatalf("reading the history: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("history = %d rows, want 1", len(actions))
	}

	got := actions[0]
	if got.Action != "removed" {
		t.Errorf("action = %q, want removed", got.Action)
	}
	if !got.ModeratorID.Valid || got.ModeratorID.UUID != f.admin {
		t.Errorf("moderator = %v, want %v", got.ModeratorID, f.admin)
	}
	if got.Note.String != "prohibited species" {
		t.Errorf("note = %q, want the reason it was removed for", got.Note.String)
	}
}

func TestDismissingKeepsTheListingVisible(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	f.reportBy(t, f.buyer)

	listing, resolved, err := f.mod.Moderate(ctx, f.admin, f.listing, "dismiss", "nothing wrong with it")
	if err != nil {
		t.Fatalf("dismissing: %v", err)
	}

	if listing.RemovedAt.Valid {
		t.Error("dismissing a report removed the listing")
	}
	if resolved != 1 {
		t.Errorf("resolved = %d, want 1", resolved)
	}

	rows, err := f.db.ListReportsForListing(ctx, f.listing)
	if err != nil {
		t.Fatalf("listing reports: %v", err)
	}
	if rows[0].Status != "dismissed" {
		t.Errorf("report = %q, want dismissed", rows[0].Status)
	}
}

func TestARemovedListingLeavesEveryReadPath(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	f.remove(t)

	browse, err := f.listings.SearchListings(ctx, uuid.Nil, dtos.ListingSearchQuery{})
	if err != nil {
		t.Fatalf("browsing: %v", err)
	}
	for _, l := range browse.Items {
		if l.ID == f.listing {
			t.Error("a removed listing is still in the browse list")
		}
	}

	found, err := f.listings.SearchListings(ctx, uuid.Nil, dtos.ListingSearchQuery{Keyword: "Chanterelles"})
	if err != nil {
		t.Fatalf("searching: %v", err)
	}
	for _, l := range found.Items {
		if l.ID == f.listing {
			t.Error("a removed listing is still in the search results")
		}
	}

	if _, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
		ListingID: f.listing, Quantity: 1,
	}); !isNotFound(err) {
		t.Errorf("ordering a removed listing: err = %#v, want *NotFoundError", err)
	}

	if _, _, err := f.chat.StartConversation(ctx, f.buyer, f.listing, "still available?"); !isNotFound(err) {
		t.Errorf("messaging about a removed listing: err = %#v, want *NotFoundError", err)
	}

	if err := f.saved.SaveListing(ctx, f.buyer, f.listing); !isNotFound(err) {
		t.Errorf("saving a removed listing: err = %#v, want *NotFoundError", err)
	}
}

func TestARemovedListingDropsOutOfTheWishlist(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	if err := f.saved.SaveListing(ctx, f.buyer, f.listing); err != nil {
		t.Fatalf("saving: %v", err)
	}

	f.remove(t)

	saved, err := f.saved.ListSaved(ctx, f.buyer)
	if err != nil {
		t.Fatalf("listing the wishlist: %v", err)
	}
	for _, l := range saved {
		if l.ID == f.listing {
			t.Error("a listing saved before removal is still in the wishlist, and carries removed_at")
		}
	}
}

func TestTheSellerCannotDeleteAModeratorRemovedListing(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	f.reportBy(t, f.buyer)
	f.remove(t)

	err := f.listings.DeleteListing(ctx, f.seller, f.listing)

	var forbidden *ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Fatalf("err = %#v, want *ForbiddenError - deleting it would erase the reports and the audit trail", err)
	}

	reports, err := f.db.ListReportsForListing(ctx, f.listing)
	if err != nil {
		t.Fatalf("listing reports: %v", err)
	}
	if len(reports) != 1 {
		t.Errorf("reports = %d, want 1 - the evidence must survive", len(reports))
	}

	actions, err := f.mod.History(ctx, f.listing)
	if err != nil {
		t.Fatalf("reading the history: %v", err)
	}
	if len(actions) != 1 {
		t.Errorf("audit rows = %d, want 1", len(actions))
	}
}

func TestReportingARemovedListingLooksLikeReportingANonexistentOne(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	f.remove(t)

	removed := f.reports.Report(ctx, f.buyer, f.listing, "prohibited", "")
	if !isNotFound(removed) {
		t.Fatalf("reporting a removed listing: err = %#v, want *NotFoundError", removed)
	}

	missing := f.reports.Report(ctx, f.buyer, database.NewID(), "prohibited", "")
	if !isNotFound(missing) {
		t.Fatalf("reporting a nonexistent listing: err = %#v, want *NotFoundError", missing)
	}

	if removed.Error() != missing.Error() {
		t.Errorf("the two refusals differ: %q then %q - that tells the reporter it was moderated", removed, missing)
	}
}

func TestTheSellerCanStillFetchTheirRemovedListing(t *testing.T) {
	f := newModerationFixture(t)

	f.remove(t)

	listing, err := f.listings.GetListing(context.Background(), f.listing)
	if err != nil {
		t.Fatalf("the seller cannot reach their own removed listing: %v", err)
	}
	if !listing.RemovedAt.Valid {
		t.Error("removed_at is not set, so the seller cannot tell what happened")
	}
}

func TestRemovingNeedsAReason(t *testing.T) {
	f := newModerationFixture(t)

	_, _, err := f.mod.Moderate(context.Background(), f.admin, f.listing, "remove", "  ")

	var invalid *ValidationError
	if !errors.As(err, &invalid) {
		t.Errorf("err = %#v, want *ValidationError - an audit row with no reason is just a timestamp", err)
	}
}

func TestAnUnknownActionIsRefused(t *testing.T) {
	f := newModerationFixture(t)

	_, _, err := f.mod.Moderate(context.Background(), f.admin, f.listing, "escalate", "up to whom?")

	var invalid *ValidationError
	if !errors.As(err, &invalid) {
		t.Errorf("err = %#v, want *ValidationError", err)
	}
}

func TestRemovingTwiceIsAConflict(t *testing.T) {
	f := newModerationFixture(t)

	f.remove(t)

	_, _, err := f.mod.Moderate(context.Background(), f.admin, f.listing, "remove", "again")

	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Errorf("err = %#v, want *ConflictError - another moderator got there first", err)
	}
}

func TestDismissingARemovedListingIsAConflict(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	f.reportBy(t, f.buyer)
	f.remove(t)

	_, _, err := f.mod.Moderate(ctx, f.admin, f.listing, "dismiss", "looks fine to me")

	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %#v, want *ConflictError - dismissing a removed listing leaves it invisible", err)
	}

	listing, err := f.db.GetListing(ctx, f.listing)
	if err != nil {
		t.Fatalf("re-reading the listing: %v", err)
	}
	if !listing.RemovedAt.Valid {
		t.Error("the refused dismiss changed the listing")
	}

	actions, err := f.mod.History(ctx, f.listing)
	if err != nil {
		t.Fatalf("reading the history: %v", err)
	}
	if len(actions) != 1 {
		t.Errorf("audit rows = %d, want 1 - a refused action must not be logged", len(actions))
	}
}

func TestRestoreBringsTheListingBack(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	f.remove(t)

	listing, _, err := f.mod.Moderate(ctx, f.admin, f.listing, "restore", "reported in error")
	if err != nil {
		t.Fatalf("restoring: %v", err)
	}
	if listing.RemovedAt.Valid {
		t.Fatal("the listing is still marked removed")
	}

	browse, err := f.listings.SearchListings(ctx, uuid.Nil, dtos.ListingSearchQuery{})
	if err != nil {
		t.Fatalf("browsing: %v", err)
	}
	var back bool
	for _, l := range browse.Items {
		if l.ID == f.listing {
			back = true
		}
	}
	if !back {
		t.Error("a restored listing did not come back to the browse list")
	}

	actions, err := f.mod.History(ctx, f.listing)
	if err != nil {
		t.Fatalf("reading the history: %v", err)
	}
	if len(actions) != 2 {
		t.Errorf("history = %d rows, want 2 - the removal is not erased by the restore", len(actions))
	}
}

func TestAnOrderPlacedBeforeRemovalStaysCancellable(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	order, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
		ListingID: f.listing, Quantity: 2,
	})
	if err != nil {
		t.Fatalf("ordering: %v", err)
	}

	f.remove(t)

	cancelled, err := f.orders.CancelOrder(ctx, f.buyer, order.ID)
	if err != nil {
		t.Fatalf("cancelling an order whose listing was removed: %v", err)
	}
	if cancelled.Status != "cancelled" {
		t.Errorf("status = %q, want cancelled", cancelled.Status)
	}

	listing, err := f.db.GetListing(ctx, f.listing)
	if err != nil {
		t.Fatalf("re-reading the listing: %v", err)
	}
	if listing.Quantity != 5 {
		t.Errorf("quantity = %d, want 5 - cancelling must still return the stock", listing.Quantity)
	}
}

// Option C from #147: removal stops the bytes being served, restore brings
// them back. Hiding the listing while its photos stayed fetchable left the
// content that caused the removal reachable to anyone holding the URL.
func TestRemovalQuarantinesTheImagesAndRestoreReleasesThem(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	name, err := f.files.Save(strings.NewReader("a photo"), ".jpg")
	if err != nil {
		t.Fatalf("saving an image: %v", err)
	}
	if _, err := f.db.CreateListingImage(ctx, database.CreateListingImageParams{
		ID:        database.NewID(),
		ListingID: f.listing,
		Filename:  name,
	}); err != nil {
		t.Fatalf("recording the image: %v", err)
	}

	served := func() bool {
		_, err := os.Stat(filepath.Join(f.files.Dir(), name))
		return err == nil
	}

	if !served() {
		t.Fatal("the image was not where it is served from")
	}

	f.remove(t)
	if served() {
		t.Error("a removed listing's image is still served")
	}

	if _, _, err := f.mod.Moderate(ctx, f.admin, f.listing, "restore", "reported in error"); err != nil {
		t.Fatalf("restoring: %v", err)
	}
	if !served() {
		t.Error("restoring the listing did not bring its image back")
	}

	// The rows are what a restore uses to find the files, so they must survive.
	images, err := f.db.ListListingImages(ctx, f.listing)
	if err != nil {
		t.Fatalf("listing images: %v", err)
	}
	if len(images) != 1 {
		t.Errorf("listing_images rows = %d, want 1", len(images))
	}
}

func TestDismissLeavesTheImagesAlone(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	name, err := f.files.Save(strings.NewReader("a photo"), ".jpg")
	if err != nil {
		t.Fatalf("saving an image: %v", err)
	}
	if _, err := f.db.CreateListingImage(ctx, database.CreateListingImageParams{
		ID: database.NewID(), ListingID: f.listing, Filename: name,
	}); err != nil {
		t.Fatalf("recording the image: %v", err)
	}

	f.reportBy(t, f.buyer)
	if _, _, err := f.mod.Moderate(ctx, f.admin, f.listing, "dismiss", "nothing wrong"); err != nil {
		t.Fatalf("dismissing: %v", err)
	}

	if _, err := os.Stat(filepath.Join(f.files.Dir(), name)); err != nil {
		t.Errorf("dismissing a report moved the listing's image: %v", err)
	}
}

// Both read paths, because they are siblings on the same resource: fixing one
// and not the other is the inconsistency this is here to remove.
func TestTheReadPathsOfAListingThatDoesNotExistAreNotFound(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	if _, err := f.mod.History(ctx, database.NewID()); !isNotFound(err) {
		t.Errorf("history: err = %v, want NotFoundError", err)
	}
	if _, err := f.mod.ReportsFor(ctx, database.NewID()); !isNotFound(err) {
		t.Errorf("reports: err = %v, want NotFoundError", err)
	}
}

// A 404 test cannot tell "no such listing" from "the database is down". This
// one uses the fixture's real listing, so only the distinction can pass it.
func TestADatabaseFailureIsNotReportedAsAMissingListing(t *testing.T) {
	f := newModerationFixture(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.mod.History(ctx, f.listing)

	if isNotFound(err) {
		t.Fatalf("err = %v, want the underlying failure rather than a 404", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the context cancellation to survive", err)
	}
}

func TestRemovingAListingTellsItsSeller(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	listing, err := f.db.GetListing(ctx, f.listing)
	if err != nil {
		t.Fatalf("reading the listing: %v", err)
	}

	if _, _, err := f.mod.Moderate(ctx, f.admin, f.listing, "remove", "not a foraged good"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	got, err := f.db.ListNotifications(ctx, database.ListNotificationsParams{UserID: f.seller, Limit: 30})
	if err != nil {
		t.Fatalf("the seller's notifications: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("the seller has %d notifications, want 1", len(got))
	}
	if got[0].Kind != notifyKindListingRemoved {
		t.Errorf("kind = %q, want %q", got[0].Kind, notifyKindListingRemoved)
	}
	if !got[0].ListingID.Valid || got[0].ListingID.UUID != f.listing {
		t.Errorf("listing = %v, want %v", got[0].ListingID, f.listing)
	}
	// The title is snapshotted so the inbox can still name the listing once
	// nothing else will show it - a removed listing is not in any index.
	if got[0].ListingTitle.String != listing.Title {
		t.Errorf("title = %q, want %q", got[0].ListingTitle.String, listing.Title)
	}
}

func TestRestoringAndDismissingTellNobody(t *testing.T) {
	for _, action := range []string{"restore", "dismiss"} {
		t.Run(action, func(t *testing.T) {
			f := newModerationFixture(t)
			ctx := context.Background()

			// restore needs something to restore; dismiss needs a listing that
			// is not removed, which is the fixture's starting state.
			if action == "restore" {
				if _, _, err := f.mod.Moderate(ctx, f.admin, f.listing, "remove", "reason"); err != nil {
					t.Fatalf("setting up the removal: %v", err)
				}
			}

			before, err := f.db.ListNotifications(ctx, database.ListNotificationsParams{UserID: f.seller, Limit: 30})
			if err != nil {
				t.Fatalf("notifications: %v", err)
			}

			if _, _, err := f.mod.Moderate(ctx, f.admin, f.listing, action, "reason"); err != nil {
				t.Fatalf("%s: %v", action, err)
			}

			after, err := f.db.ListNotifications(ctx, database.ListNotificationsParams{UserID: f.seller, Limit: 30})
			if err != nil {
				t.Fatalf("notifications: %v", err)
			}
			if len(after) != len(before) {
				t.Errorf("%s wrote %d notifications, want 0 - there is no kind for it", action, len(after)-len(before))
			}
		})
	}
}

func TestTheRemovalRollsBackWhenTheNotificationFails(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	if _, err := f.db.Exec("ALTER TABLE notifications RENAME TO notifications_hidden"); err != nil {
		t.Fatalf("hiding the notifications table: %v", err)
	}

	if _, _, err := f.mod.Moderate(ctx, f.admin, f.listing, "remove", "not a foraged good"); err == nil {
		t.Fatal("remove succeeded despite the notification write failing")
	}

	if _, err := f.db.Exec("ALTER TABLE notifications_hidden RENAME TO notifications"); err != nil {
		t.Fatalf("restoring the notifications table: %v", err)
	}

	listing, err := f.db.GetListing(ctx, f.listing)
	if err != nil {
		t.Fatalf("re-reading the listing: %v", err)
	}
	if listing.RemovedAt.Valid {
		t.Error("the removal outlived its failed notification")
	}
}

// The delete leg of saved_listing_gone. This is the test that catches the
// cascade: notifications.listing_id is a foreign key with ON DELETE CASCADE,
// so a row pointing at the listing being deleted is erased inside the same
// transaction and the savers are told nothing.
func TestDeletingAListingTellsEveryoneWhoSavedIt(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	for _, saver := range []uuid.UUID{f.buyer, f.other} {
		if err := f.saved.SaveListing(ctx, saver, f.listing); err != nil {
			t.Fatalf("saving as %v: %v", saver, err)
		}
	}
	// The seller saved their own listing, and is the one deleting it.
	if err := f.saved.SaveListing(ctx, f.seller, f.listing); err != nil {
		t.Fatalf("the seller saving their own: %v", err)
	}

	if err := f.listings.DeleteListing(ctx, f.seller, f.listing); err != nil {
		t.Fatalf("delete: %v", err)
	}

	for _, saver := range []uuid.UUID{f.buyer, f.other} {
		got, err := f.db.ListNotifications(ctx, database.ListNotificationsParams{UserID: saver, Limit: 30})
		if err != nil {
			t.Fatalf("notifications for %v: %v", saver, err)
		}
		var found *database.Notification
		for i := range got {
			if got[i].Kind == notifyKindSavedListingDeleted {
				found = &got[i]
			}
		}
		if found == nil {
			t.Fatalf("%v was not told their saved listing was deleted", saver)
		}
		// The seller, because the listing no longer exists to point at.
		if !found.ActorID.Valid || found.ActorID.UUID != f.seller {
			t.Errorf("actor = %v, want the seller %v", found.ActorID, f.seller)
		}
		if found.ListingID.Valid {
			t.Error("the row points at the deleted listing, so the cascade will have eaten it")
		}
		if found.ListingTitle.String == "" {
			t.Error("no title snapshot, so the inbox cannot name what is gone")
		}
	}

	sellers, err := f.db.ListNotifications(ctx, database.ListNotificationsParams{UserID: f.seller, Limit: 30})
	if err != nil {
		t.Fatalf("the seller's notifications: %v", err)
	}
	for _, n := range sellers {
		if n.Kind == notifyKindSavedListingDeleted {
			t.Error("the seller was told about their own deletion")
		}
	}
}

func TestTheDeleteRollsBackWhenTheNotificationFails(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	if err := f.saved.SaveListing(ctx, f.buyer, f.listing); err != nil {
		t.Fatalf("saving: %v", err)
	}
	if _, err := f.db.Exec("ALTER TABLE notifications RENAME TO notifications_hidden"); err != nil {
		t.Fatalf("hiding the notifications table: %v", err)
	}

	if err := f.listings.DeleteListing(ctx, f.seller, f.listing); err == nil {
		t.Fatal("delete succeeded despite the notification write failing")
	}

	if _, err := f.db.Exec("ALTER TABLE notifications_hidden RENAME TO notifications"); err != nil {
		t.Fatalf("restoring the notifications table: %v", err)
	}

	if _, err := f.db.GetListing(ctx, f.listing); err != nil {
		t.Errorf("the listing was deleted despite its failed notification: %v", err)
	}
}

// The sell-out leg. Unlike a deletion the listing still exists, so the row
// points at it and the reader lands on the listing itself.
func TestSellingOutTellsEveryoneWhoSavedIt(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	for _, saver := range []uuid.UUID{f.other, f.buyer} {
		if err := f.saved.SaveListing(ctx, saver, f.listing); err != nil {
			t.Fatalf("saving as %v: %v", saver, err)
		}
	}

	// Five is the fixture's whole stock, so this order empties it.
	if _, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
		ListingID: f.listing, Quantity: 5,
	}); err != nil {
		t.Fatalf("buying the lot: %v", err)
	}

	got, err := f.db.ListNotifications(ctx, database.ListNotificationsParams{UserID: f.other, Limit: 30})
	if err != nil {
		t.Fatalf("notifications: %v", err)
	}
	var found *database.Notification
	for i := range got {
		if got[i].Kind == notifyKindSavedListingGone {
			found = &got[i]
		}
	}
	if found == nil {
		t.Fatal("a saver was not told the listing sold out")
	}
	if !found.ListingID.Valid || found.ListingID.UUID != f.listing {
		t.Errorf("listing = %v, want %v - the listing still exists, so point at it", found.ListingID, f.listing)
	}

	// The buyer emptied it themselves and does not need telling.
	buyers, err := f.db.ListNotifications(ctx, database.ListNotificationsParams{UserID: f.buyer, Limit: 30})
	if err != nil {
		t.Fatalf("the buyer's notifications: %v", err)
	}
	for _, n := range buyers {
		if n.Kind == notifyKindSavedListingGone {
			t.Error("the buyer who emptied the stock was told it was gone")
		}
	}
}

func TestAPartialPurchaseTellsNobody(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	if err := f.saved.SaveListing(ctx, f.other, f.listing); err != nil {
		t.Fatalf("saving: %v", err)
	}

	if _, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
		ListingID: f.listing, Quantity: 4,
	}); err != nil {
		t.Fatalf("buying four of five: %v", err)
	}

	got, err := f.db.ListNotifications(ctx, database.ListNotificationsParams{UserID: f.other, Limit: 30})
	if err != nil {
		t.Fatalf("notifications: %v", err)
	}
	for _, n := range got {
		if n.Kind == notifyKindSavedListingGone {
			t.Error("a saver was told the listing was gone while one is still in stock")
		}
	}
}

// A sold-out notice is only true while the stock is gone. Cancelling puts it
// back, so the notice has to go with it - otherwise the reader is sent to an
// in-stock listing, and a repeated buy-and-cancel fills their inbox with
// copies until the 30-row cap has pushed everything real out of it.
func TestCancellingAnOrderClearsTheSoldOutNotice(t *testing.T) {
	f := newModerationFixture(t)
	ctx := context.Background()

	if err := f.saved.SaveListing(ctx, f.other, f.listing); err != nil {
		t.Fatalf("saving: %v", err)
	}

	soldOut := func() int {
		t.Helper()
		got, err := f.db.ListNotifications(ctx, database.ListNotificationsParams{UserID: f.other, Limit: 30})
		if err != nil {
			t.Fatalf("notifications: %v", err)
		}
		n := 0
		for _, row := range got {
			if row.Kind == notifyKindSavedListingGone {
				n++
			}
		}
		return n
	}

	for round := 1; round <= 3; round++ {
		order, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
			ListingID: f.listing, Quantity: 5,
		})
		if err != nil {
			t.Fatalf("round %d, buying the lot: %v", round, err)
		}
		if got := soldOut(); got != 1 {
			t.Fatalf("round %d: sold-out notices = %d, want 1", round, got)
		}

		if _, err := f.orders.CancelOrder(ctx, f.seller, order.ID); err != nil {
			t.Fatalf("round %d, cancelling: %v", round, err)
		}

		if got := soldOut(); got != 0 {
			t.Errorf("round %d: %d sold-out notices survived the restock", round, got)
		}

		listing, err := f.db.GetListing(ctx, f.listing)
		if err != nil {
			t.Fatalf("re-reading the listing: %v", err)
		}
		if listing.Quantity != 5 {
			t.Fatalf("round %d: quantity = %d, want 5 back in stock", round, listing.Quantity)
		}
	}
}
