package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

type exportFixture struct {
	users  *UserService
	db     *database.DB
	rec    *recorder
	me     uuid.UUID
	other  uuid.UUID
	myMail string
}

func newExportFixture(t *testing.T) exportFixture {
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
		if err := db.EnsureProfile(ctx, user.ID); err != nil {
			t.Fatalf("creating %s's profile: %v", name, err)
		}
		return user.ID
	}
	me, other := mk("me"), mk("other")

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	rec := &recorder{}
	return exportFixture{
		users:  NewUserService(db, files, rec),
		db:     db,
		rec:    rec,
		me:     me,
		other:  other,
		myMail: "me@example.test",
	}
}

func (f exportFixture) export(t *testing.T) dtos.DataExport {
	t.Helper()

	out, err := f.users.ExportData(context.Background(), f.me)
	if err != nil {
		t.Fatalf("exporting: %v", err)
	}
	return out
}

// The listing filters the buyer-facing search applies are absent on purpose:
// this is the seller's record of what they wrote, so a sold-out or removed
// listing is theirs too.
func TestTheExportCarriesSoldOutAndRemovedListings(t *testing.T) {
	f := newExportFixture(t)
	ctx := context.Background()

	for _, title := range []string{"In stock", "Sold out", "Removed"} {
		l, err := f.db.CreateListing(ctx, database.CreateListingParams{
			ID: database.NewID(), SellerID: f.me, Title: title,
			Category: "mushrooms", Price: "18.10", Quantity: 4, Unit: "kg",
		})
		if err != nil {
			t.Fatalf("creating %q: %v", title, err)
		}
		switch title {
		case "Sold out":
			if _, err := f.db.ExecContext(ctx, `UPDATE listings SET quantity = 0 WHERE id = $1`, l.ID); err != nil {
				t.Fatalf("selling out: %v", err)
			}
		case "Removed":
			if _, err := f.db.ExecContext(ctx, `UPDATE listings SET removed_at = now() WHERE id = $1`, l.ID); err != nil {
				t.Fatalf("removing: %v", err)
			}
		}
	}

	got := map[string]bool{}
	for _, l := range f.export(t).Listings {
		got[l.Title] = true
	}
	for _, want := range []string{"In stock", "Sold out", "Removed"} {
		if !got[want] {
			t.Errorf("export is missing %q - %v", want, got)
		}
	}
}

// A block is symmetric in its effects precisely so neither side learns who
// acted. An export naming the people who blocked you would be the disclosure
// channel the rest of the API refuses to be.
func TestTheExportNamesWhoYouBlockedAndNotWhoBlockedYou(t *testing.T) {
	f := newExportFixture(t)
	ctx := context.Background()

	blocks := NewBlockService(f.db.Queries)
	if err := blocks.Block(ctx, f.me, f.other); err != nil {
		t.Fatalf("blocking: %v", err)
	}
	if err := blocks.Block(ctx, f.other, f.me); err != nil {
		t.Fatalf("being blocked: %v", err)
	}

	out := f.export(t)
	if len(out.Blocks) != 1 {
		t.Fatalf("blocks = %+v, want the one this account made", out.Blocks)
	}
	if out.Blocks[0].ID != f.other {
		t.Errorf("blocked = %v, want %v", out.Blocks[0].ID, f.other)
	}
}

// Both sides of a conversation: the other person's text is readable in the app
// already, and half a thread is worse than none.
func TestTheExportCarriesBothSidesOfAConversation(t *testing.T) {
	f := newExportFixture(t)
	ctx := context.Background()

	listing, err := f.db.CreateListing(ctx, database.CreateListingParams{
		ID: database.NewID(), SellerID: f.other, Title: "Chanterelles",
		Category: "mushrooms", Price: "18.10", Quantity: 4, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating a listing: %v", err)
	}

	chat := NewConversationService(f.db, notify.Disabled{})
	conv, _, err := chat.StartConversation(ctx, f.me, listing.ID, "is this still available?")
	if err != nil {
		t.Fatalf("starting the conversation: %v", err)
	}
	if _, err := chat.Accept(ctx, f.other, conv.ID); err != nil {
		t.Fatalf("accepting: %v", err)
	}
	if _, err := chat.SendMessage(ctx, f.other, conv.ID, "it is"); err != nil {
		t.Fatalf("replying: %v", err)
	}

	out := f.export(t)
	if len(out.Conversations) != 1 {
		t.Fatalf("conversations = %d, want 1", len(out.Conversations))
	}
	if len(out.Messages) != 2 {
		t.Fatalf("messages = %d, want both sides", len(out.Messages))
	}
}

// Sent after the export is assembled, to the address on the account, and it
// does not carry the file.
func TestExportingSendsOneConfirmationWithoutTheData(t *testing.T) {
	f := newExportFixture(t)

	out := f.export(t)

	f.rec.only(t, notify.KindDataExported, f.myMail)

	// It confirms; it must not carry. Mailing the record would put it in an
	// inbox and in whatever relay handled it.
	body := f.rec.messages()[0].Body
	if strings.Contains(body, out.Account.Username) && strings.Contains(body, "\"listings\"") {
		t.Error("the confirmation carries the export itself")
	}
	if !strings.Contains(body, out.Account.Username) {
		t.Errorf("body = %q, want it addressed to the account", body)
	}
}

// The endpoint's purpose is "everything about this person, AND NOTHING ABOUT
// ANYONE ELSE". Every other test here asserts the first half. This is the
// second, and it covers all fifteen queries at once: change any WHERE from
// `= $1` to something wider and a stranger's rows appear in this document.
//
// One test rather than fifteen, because a copy-pasted or edited predicate is
// the failure being guarded against, and it can happen in any of them.
func TestTheExportContainsNothingBelongingToAnyoneElse(t *testing.T) {
	f := newExportFixture(t)
	ctx := context.Background()

	// A whole unrelated account: its own listing, its own order, its own
	// conversation, its own notification.
	stranger := f.other
	strangerListing, err := f.db.CreateListing(ctx, database.CreateListingParams{
		ID: database.NewID(), SellerID: stranger, Title: "Not yours",
		Category: "mushrooms", Price: "99.00", Quantity: 5, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("the stranger's listing: %v", err)
	}

	third, err := f.db.CreateUser(ctx, database.CreateUserParams{
		ID:       database.NewID(),
		Username: "third", Email: "third@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("a third account: %v", err)
	}
	if err := f.db.EnsureProfile(ctx, third.ID); err != nil {
		t.Fatalf("the third profile: %v", err)
	}

	// An order and a conversation between the two OTHER accounts, so neither
	// side of either is the exporting user.
	orders := NewOrderService(f.db, notify.Disabled{})
	strangerOrder, err := orders.CreateOrder(ctx, third.ID, dtos.CreateOrderInput{
		ListingID: strangerListing.ID, Quantity: 1,
	})
	if err != nil {
		t.Fatalf("the stranger's order: %v", err)
	}

	chat := NewConversationService(f.db, notify.Disabled{})
	strangerConv, strangerMsg, err := chat.StartConversation(ctx, third.ID, strangerListing.ID, "mine, not yours")
	if err != nil {
		t.Fatalf("the stranger's conversation: %v", err)
	}

	out := f.export(t)

	// Rendered as JSON and searched by id: a leak through any collection shows
	// up here even if that collection has no assertion of its own, which is the
	// point of doing it this way rather than field by field.
	doc, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshalling the export: %v", err)
	}

	for _, forbidden := range []struct {
		what string
		id   uuid.UUID
	}{
		{"the stranger's listing", strangerListing.ID},
		{"an order between two other accounts", strangerOrder.ID},
		{"a conversation between two other accounts", strangerConv.ID},
		{"a message in it", strangerMsg.ID},
		{"a third account's id", third.ID},
	} {
		if strings.Contains(string(doc), forbidden.id.String()) {
			t.Errorf("the export contains %s (%s)", forbidden.what, forbidden.id)
		}
	}

	// And the guard that stops the above passing vacuously: the caller's own
	// data IS in there, so the search is looking at a populated document.
	if !strings.Contains(string(doc), f.me.String()) {
		t.Fatal("the export does not contain the caller's own id, so finding nothing proves nothing")
	}
}

// A notification's subject is the only thing that makes it meaningful, and the
// export reads the table with SELECT *. When migration 022 added actor_id and
// listing_id, the generated query kept its old column list and the export
// silently published null for both - it still compiled, because the row struct
// has the fields either way. This is what notices.
func TestTheExportCarriesWhatANotificationIsAbout(t *testing.T) {
	f := newExportFixture(t)
	ctx := context.Background()

	listing, err := f.db.CreateListing(ctx, database.CreateListingParams{
		ID: database.NewID(), SellerID: f.me, Title: "Chanterelles",
		Category: "mushrooms", Price: "12.00", Quantity: 3, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("a listing: %v", err)
	}

	if err := f.db.CreateNotification(ctx, database.CreateNotificationParams{
		ID:        database.NewID(),
		UserID:    f.me,
		Kind:      "new_follower",
		ActorID:   uuid.NullUUID{UUID: f.other, Valid: true},
		ListingID: uuid.NullUUID{},
	}); err != nil {
		t.Fatalf("an actor notification: %v", err)
	}

	if err := f.db.CreateNotification(ctx, database.CreateNotificationParams{
		ID:           database.NewID(),
		UserID:       f.me,
		Kind:         "listing_removed",
		ListingTitle: sql.NullString{String: listing.Title, Valid: true},
		ListingID:    uuid.NullUUID{UUID: listing.ID, Valid: true},
	}); err != nil {
		t.Fatalf("a listing notification: %v", err)
	}

	out := f.export(t)

	if len(out.Notifications) != 2 {
		t.Fatalf("exported %d notifications, want 2", len(out.Notifications))
	}

	var sawActor, sawListing bool
	for _, n := range out.Notifications {
		if n.ActorID != nil && *n.ActorID == f.other {
			sawActor = true
		}
		if n.ListingID != nil && *n.ListingID == listing.ID {
			sawListing = true
		}
	}
	if !sawActor {
		t.Error("the exported notification does not say who acted")
	}
	if !sawListing {
		t.Error("the exported notification does not say which listing it is about")
	}
}
