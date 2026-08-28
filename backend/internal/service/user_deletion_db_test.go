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

type deletionFixture struct {
	users    *UserService
	listings *ListingService
	orders   *OrderService
	chat     *ConversationService
	profiles *ProfileService
	db       *database.DB
	files    *storage.Local
	seller   uuid.UUID
	buyer    uuid.UUID
	listing  uuid.UUID
}

func newDeletionFixture(t *testing.T) deletionFixture {
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
	seller, buyer := mk("seller"), mk("buyer")

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: seller, Title: "Chanterelles", Category: "mushrooms",
		Price: "18.10", Quantity: 10, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating a listing: %v", err)
	}

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	return deletionFixture{
		users:    NewUserService(db, files),
		listings: NewListingService(db, files),
		orders:   NewOrderService(db, notify.Disabled{}),
		chat:     NewConversationService(db, notify.Disabled{}),
		profiles: NewProfileService(db, files),
		db:       db,
		files:    files,
		seller:   seller,
		buyer:    buyer,
		listing:  listing.ID,
	}
}

// The case a hard delete cannot do at all: orders references users with
// ON DELETE RESTRICT on both sides.
func TestAUserWithOrdersCanStillBeDeleted(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	order, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
		ListingID: f.listing, Quantity: 2,
	})
	if err != nil {
		t.Fatalf("ordering: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.buyer, "buyer"); err != nil {
		t.Fatalf("deleting a buyer who has ordered: %v", err)
	}

	kept, err := f.db.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("the order did not survive: %v", err)
	}
	if kept.BuyerID != f.buyer {
		t.Error("the order lost its buyer")
	}
}

func TestDeletionScrubsTheIdentifyingFields(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	if err := f.users.DeleteAccount(ctx, f.buyer, "buyer"); err != nil {
		t.Fatalf("deleting: %v", err)
	}

	user, err := f.db.GetUser(ctx, f.buyer)
	if err != nil {
		t.Fatalf("the row did not survive: %v", err)
	}

	if !user.DeletedAt.Valid {
		t.Error("deleted_at was not set")
	}
	if user.Password.Valid {
		t.Error("the password hash survived")
	}
	if user.LastSeenAt.Valid {
		t.Error("last_seen_at survived, so presence still leaks")
	}
	if strings.Contains(user.Email, "buyer@example.test") {
		t.Errorf("the old address survived: %q", user.Email)
	}
	if !strings.HasSuffix(user.Email, "@deleted.invalid") {
		t.Errorf("email = %q, want a non-routable placeholder", user.Email)
	}
	if user.Username == "buyer" {
		t.Error("the old username survived")
	}
}

func TestTheCounterpartyKeepsTheirThread(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	conv, _, err := f.chat.StartConversation(ctx, f.buyer, f.listing, "are these fresh?")
	if err != nil {
		t.Fatalf("starting the conversation: %v", err)
	}
	if _, err := f.chat.Accept(ctx, f.seller, conv.ID); err != nil {
		t.Fatalf("accepting: %v", err)
	}
	if _, err := f.chat.SendMessage(ctx, f.seller, conv.ID, "picked this morning"); err != nil {
		t.Fatalf("seller replying: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.buyer, "buyer"); err != nil {
		t.Fatalf("deleting the buyer: %v", err)
	}

	// The whole reason the row is anonymised rather than deleted: messages
	// cascade from conversations, so a real delete would take the seller's own
	// words with it.
	inbox, err := f.chat.ListConversations(ctx, f.seller)
	if err != nil {
		t.Fatalf("listing the seller's inbox: %v", err)
	}
	if len(inbox) != 1 {
		t.Fatalf("the seller sees %d threads, want 1", len(inbox))
	}
	if !inbox[0].OtherDeletedAt.Valid {
		t.Error("the thread does not know the other party is gone")
	}

	msgs, err := f.chat.ListMessages(ctx, f.seller, conv.ID, uuid.Nil, 0)
	if err != nil {
		t.Fatalf("listing messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("messages = %d, want 2 - the seller's own words must survive", len(msgs))
	}

	if _, err := f.chat.SendMessage(ctx, f.seller, conv.ID, "still there?"); err == nil {
		t.Error("the seller could still message a deleted user")
	}
}

func TestDeletionRemovesTheAvatarFile(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	name, err := f.files.Save(strings.NewReader("a face"), ".jpg")
	if err != nil {
		t.Fatalf("saving an avatar: %v", err)
	}
	if _, err := f.db.SetAvatar(ctx, database.SetAvatarParams{
		ID:             f.buyer,
		AvatarFilename: sql.NullString{String: name, Valid: true},
	}); err != nil {
		t.Fatalf("recording the avatar: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.buyer, "buyer"); err != nil {
		t.Fatalf("deleting: %v", err)
	}

	if _, err := os.Stat(filepath.Join(f.files.Dir(), name)); !os.IsNotExist(err) {
		t.Errorf("the avatar file survived the deletion: %v", err)
	}
}

func TestAWrongConfirmationChangesNothing(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	var invalid *ValidationError

	// Case and whitespace must not pass: a confirmation that accepts near
	// misses is not a confirmation.
	for _, wrong := range []string{"", "Buyer", "buyer ", "seller"} {
		if err := f.users.DeleteAccount(ctx, f.buyer, wrong); !errors.As(err, &invalid) {
			t.Errorf("confirmation %q: err = %#v, want *ValidationError", wrong, err)
		}
	}

	user, err := f.db.GetUser(ctx, f.buyer)
	if err != nil {
		t.Fatalf("re-reading: %v", err)
	}
	if user.DeletedAt.Valid || user.Username != "buyer" {
		t.Error("a refused deletion still changed the account")
	}
}

func TestDeletingTwiceIsANotFound(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	if err := f.users.DeleteAccount(ctx, f.buyer, "buyer"); err != nil {
		t.Fatalf("first deletion: %v", err)
	}

	err := f.users.DeleteAccount(ctx, f.buyer, "buyer")

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("err = %#v, want *NotFoundError", err)
	}
}

func TestADeletedSellersListingsLeaveEveryReadPath(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	if err := f.users.DeleteAccount(ctx, f.seller, "seller"); err != nil {
		t.Fatalf("deleting the seller: %v", err)
	}

	browse, err := f.listings.ListListings(ctx)
	if err != nil {
		t.Fatalf("browsing: %v", err)
	}
	for _, l := range browse {
		if l.ID == f.listing {
			t.Error("a deleted seller's listing is still in the browse list")
		}
	}

	found, err := f.listings.SearchListings(ctx, dtos.ListingSearchQuery{Keyword: "Chanterelles"})
	if err != nil {
		t.Fatalf("searching: %v", err)
	}
	for _, l := range found.Items {
		if l.ID == f.listing {
			t.Error("a deleted seller's listing is still in the search results")
		}
	}

	if _, err := f.listings.GetListing(ctx, f.listing); err == nil {
		t.Error("a deleted seller's listing can still be fetched by id")
	}

	if _, err := f.profiles.Get(ctx, f.seller); err == nil {
		t.Error("a deleted user's profile is still readable")
	}
}
