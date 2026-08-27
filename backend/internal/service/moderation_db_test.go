package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
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

	return moderationFixture{
		mod:      NewModerationService(db),
		reports:  NewReportService(db.Queries),
		orders:   NewOrderService(db),
		chat:     NewConversationService(db),
		saved:    NewSavedListingService(db.Queries),
		listings: NewListingService(db, nil),
		db:       db,
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

	browse, err := f.listings.ListListings(ctx)
	if err != nil {
		t.Fatalf("browsing: %v", err)
	}
	for _, l := range browse {
		if l.ID == f.listing {
			t.Error("a removed listing is still in the browse list")
		}
	}

	found, err := f.listings.SearchListings(ctx, dtos.ListingSearchQuery{Keyword: "Chanterelles"})
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

	browse, err := f.listings.ListListings(ctx)
	if err != nil {
		t.Fatalf("browsing: %v", err)
	}
	var back bool
	for _, l := range browse {
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
