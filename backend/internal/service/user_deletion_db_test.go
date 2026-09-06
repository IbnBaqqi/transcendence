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
	saved    *SavedListingService
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
		users:    NewUserService(db, files, notify.Disabled{}),
		listings: NewListingService(db, files),
		orders:   NewOrderService(db, notify.Disabled{}),
		chat:     NewConversationService(db, notify.Disabled{}),
		profiles: NewProfileService(db, files),
		saved:    NewSavedListingService(db.Queries),
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

	browse, err := f.listings.SearchListings(ctx, uuid.Nil, dtos.ListingSearchQuery{})
	if err != nil {
		t.Fatalf("browsing: %v", err)
	}
	for _, l := range browse.Items {
		if l.ID == f.listing {
			t.Error("a deleted seller's listing is still in the browse list")
		}
	}

	found, err := f.listings.SearchListings(ctx, uuid.Nil, dtos.ListingSearchQuery{Keyword: "Chanterelles"})
	if err != nil {
		t.Fatalf("searching: %v", err)
	}
	for _, l := range found.Items {
		if l.ID == f.listing {
			t.Error("a deleted seller's listing is still in the search results")
		}
	}

	// GetListing deliberately still returns it. Every service reads through
	// that query, and filtering it would 404 a reported listing for the admin
	// judging the report. Public visibility is enforced at the handler - see
	// TestADepartedSellersListingIsHiddenFromEveryoneButAdmins.
	if _, err := f.listings.GetListing(ctx, f.listing); err != nil {
		t.Errorf("the shared getter should still resolve it for moderation: %v", err)
	}

	if _, err := f.profiles.Get(ctx, f.seller); err == nil {
		t.Error("a deleted user's profile is still readable")
	}
}

// Everything the scrub touches beyond the users row. Each of these is a
// separate statement in DeleteAccount's step list, and none of them was
// asserted before - a refactor dropping one would have gone unnoticed.
func TestDeletionClearsTheOwnedRows(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	if err := f.db.SaveListing(ctx, database.SaveListingParams{
		UserID: f.buyer, ListingID: f.listing,
	}); err != nil {
		t.Fatalf("saving: %v", err)
	}
	if _, err := f.db.FollowUser(ctx, database.FollowUserParams{
		FollowerID: f.buyer, FolloweeID: f.seller,
	}); err != nil {
		t.Fatalf("following: %v", err)
	}
	if err := f.db.LinkIdentity(ctx, database.LinkIdentityParams{
		Provider: "github", ProviderUserID: "gh-1", UserID: f.buyer,
	}); err != nil {
		t.Fatalf("linking: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.buyer, "buyer"); err != nil {
		t.Fatalf("deleting: %v", err)
	}

	profile, err := f.db.GetProfile(ctx, f.buyer)
	if err != nil {
		t.Fatalf("reading the profile: %v", err)
	}
	if profile.Firstname.Valid || profile.Bio.Valid || profile.PhoneNumber.Valid || profile.DateOfBirth.Valid {
		t.Errorf("profile fields survived: %+v", profile)
	}

	saved, err := f.db.ListSavedListings(ctx, f.buyer)
	if err != nil {
		t.Fatalf("listing saved: %v", err)
	}
	if len(saved) != 0 {
		t.Errorf("saved listings = %d, want 0", len(saved))
	}

	following, err := f.db.ListFollowing(ctx, database.ListFollowingParams{
		ViewerID: f.buyer, SubjectID: f.buyer,
	})
	if err != nil {
		t.Fatalf("listing follows: %v", err)
	}
	if len(following) != 0 {
		t.Errorf("follows = %d, want 0", len(following))
	}

	providers, err := f.db.ListProvidersForUser(ctx, f.buyer)
	if err != nil {
		t.Fatalf("listing providers: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("oauth identities = %d, want 0", len(providers))
	}
}

// The spec promises reports and moderation actions survive "with your id
// detached". ON DELETE SET NULL cannot deliver that, because nothing is ever
// deleted - it has to be explicit.
func TestDeletionDetachesTheAuthorIds(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	reports := NewReportService(f.db.Queries)
	if err := reports.Report(ctx, f.buyer, f.listing, "spam", ""); err != nil {
		t.Fatalf("reporting: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.buyer, "buyer"); err != nil {
		t.Fatalf("deleting: %v", err)
	}

	rows, err := f.db.ListReportsForListing(ctx, f.listing)
	if err != nil {
		t.Fatalf("listing reports: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("reports = %d, want 1 - the report itself must survive", len(rows))
	}
	if rows[0].ReporterID.Valid {
		t.Error("the report still names its reporter")
	}
}

func TestADepartedSellerCannotBeOrderedFromOrMessaged(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	if err := f.users.DeleteAccount(ctx, f.seller, "seller"); err != nil {
		t.Fatalf("deleting the seller: %v", err)
	}

	if _, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
		ListingID: f.listing, Quantity: 1,
	}); !isNotFound(err) {
		t.Errorf("ordering from a departed seller: err = %#v, want *NotFoundError", err)
	}

	if _, _, err := f.chat.StartConversation(ctx, f.buyer, f.listing, "hello?"); !isNotFound(err) {
		t.Errorf("messaging a departed seller: err = %#v, want *NotFoundError", err)
	}

	if err := f.saved.SaveListing(ctx, f.buyer, f.listing); !isNotFound(err) {
		t.Errorf("saving a departed seller's listing: err = %#v, want *NotFoundError", err)
	}
}

// A block belongs to the person who made it, not to the person it hides.
func TestDeletingDoesNotUndoSomeoneElsesBlock(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	blocks := NewBlockService(f.db.Queries)
	if err := blocks.Block(ctx, f.seller, f.buyer); err != nil {
		t.Fatalf("blocking: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.buyer, "buyer"); err != nil {
		t.Fatalf("deleting the blocked user: %v", err)
	}

	still, err := blocks.ExistsBetween(ctx, f.seller, f.buyer)
	if err != nil {
		t.Fatalf("checking the block: %v", err)
	}
	if !still {
		t.Error("the seller's block died with the person it hid")
	}
}

// Deletion has to hold, not just happen. mw.TouchLastSeen runs at the
// /api/v1 level - outside the group RequireActiveUser guards - so a token
// that outlived its account writes last_seen_at straight back into the
// scrubbed row, and the counterparty sees "Deleted user" showing as online.
func TestTheScrubStaysScrubbed(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	if err := f.users.DeleteAccount(ctx, f.buyer, "buyer"); err != nil {
		t.Fatalf("deleting: %v", err)
	}

	if err := f.db.TouchLastSeen(ctx, f.buyer); err != nil {
		t.Fatalf("touching: %v", err)
	}

	user, err := f.db.GetUser(ctx, f.buyer)
	if err != nil {
		t.Fatalf("re-reading: %v", err)
	}

	if user.LastSeenAt.Valid {
		t.Error("a request re-wrote last_seen_at on a deleted account, so it reads as online")
	}
	if user.ShowOnlineStatus {
		t.Error("show_online_status survived the scrub")
	}
}

// The scrub rewrites users.email, so the recipient has to be read BEFORE the
// transaction. Look it up afterwards - the way every other notification does -
// and this arrives at deleted-<id>@example.invalid.
func TestDeletionEmailsTheAddressTheAccountHadBeforeTheScrub(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()

	user, err := db.CreateUser(ctx, database.CreateUserParams{
		ID:       database.NewID(),
		Username: "departing", Email: "departing@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating the account: %v", err)
	}
	if err := db.EnsureProfile(ctx, user.ID); err != nil {
		t.Fatalf("creating the profile: %v", err)
	}

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	rec := &recorder{}
	if err := NewUserService(db, files, rec).DeleteAccount(ctx, user.ID, "departing"); err != nil {
		t.Fatalf("deleting: %v", err)
	}

	rec.only(t, notify.KindAccountDeleted, "departing@example.test")

	// And the row really was scrubbed, so the address above could only have
	// come from before the transaction.
	after, err := db.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("reading the scrubbed row: %v", err)
	}
	if after.Email == "departing@example.test" {
		t.Fatal("the account was not scrubbed, so this proves nothing")
	}
}

// #274. A departing account used to leave its listings in the table and its
// orders at pending, with a counterparty waiting on a handover from somebody
// who no longer exists - and admin_orders.stranded does not catch that, because
// it needs BOTH sides deleted.
func TestDeletingASellerCancelsTheirOrdersAndRemovesTheirListings(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	order, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
		ListingID: f.listing, Quantity: 2,
	})
	if err != nil {
		t.Fatalf("ordering: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.seller, "seller"); err != nil {
		t.Fatalf("deleting the seller: %v", err)
	}

	t.Run("the buyer's order is cancelled rather than left pending", func(t *testing.T) {
		got, err := f.db.GetOrder(ctx, order.ID)
		if err != nil {
			t.Fatalf("re-reading the order: %v", err)
		}
		if got.Status != "cancelled" {
			t.Errorf("status = %q, want cancelled", got.Status)
		}
	})

	t.Run("and the buyer is told why their order ended", func(t *testing.T) {
		notes, err := f.db.ListNotifications(ctx, database.ListNotificationsParams{
			UserID: f.buyer, Limit: 30,
		})
		if err != nil {
			t.Fatalf("reading notifications: %v", err)
		}
		var cancelled int
		for _, n := range notes {
			if n.Kind == "order_cancelled" {
				cancelled++
			}
		}
		if cancelled != 1 {
			t.Errorf("order_cancelled notifications = %d, want 1", cancelled)
		}
	})

	t.Run("the listing row is gone, not merely hidden", func(t *testing.T) {
		if _, err := f.db.GetListing(ctx, f.listing); !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("GetListing err = %v, want sql.ErrNoRows", err)
		}
	})

	// #264: the order keeps its own snapshot, so losing the listing does not
	// cost the buyer their record of what they bought.
	t.Run("the buyer keeps their receipt", func(t *testing.T) {
		got, err := f.db.GetOrder(ctx, order.ID)
		if err != nil {
			t.Fatalf("re-reading the order: %v", err)
		}
		if got.ListingID.Valid {
			t.Errorf("listing_id = %v, want null", got.ListingID.UUID)
		}
		if got.ListingTitle != "Chanterelles" {
			t.Errorf("listing_title = %q, want Chanterelles", got.ListingTitle)
		}
	})
}

// The other direction, and the one where the restock is not a wasted write: the
// seller stays, so their listing has to get its stock back or a departing buyer
// silently costs them two kilos.
func TestDeletingABuyerGivesTheSellerTheirStockBack(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	before, err := f.db.GetListing(ctx, f.listing)
	if err != nil {
		t.Fatalf("reading the listing: %v", err)
	}

	if _, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
		ListingID: f.listing, Quantity: 2,
	}); err != nil {
		t.Fatalf("ordering: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.buyer, "buyer"); err != nil {
		t.Fatalf("deleting the buyer: %v", err)
	}

	after, err := f.db.GetListing(ctx, f.listing)
	if err != nil {
		t.Fatalf("the seller's listing should survive the buyer leaving: %v", err)
	}
	if after.Quantity != before.Quantity {
		t.Errorf("quantity = %d, want %d - the cancelled order's stock was not returned",
			after.Quantity, before.Quantity)
	}
}

// listing_reports and moderation_actions both CASCADE from listings, so
// deleting a moderator-removed listing takes the report and the record of the
// decision with it. Deleting an account must not be a way to launder that.
func TestDeletingAnAccountKeepsAModeratorRemovedListing(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	if _, err := f.db.ExecContext(ctx,
		`UPDATE listings SET removed_at = now() WHERE id = $1`, f.listing,
	); err != nil {
		t.Fatalf("marking the listing removed: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.seller, "seller"); err != nil {
		t.Fatalf("deleting the seller: %v", err)
	}

	if _, err := f.db.GetListing(ctx, f.listing); err != nil {
		t.Errorf("the moderator-removed listing is gone (%v) - its report and the "+
			"moderation decision cascade with it", err)
	}
}

// Finished orders are records. Cancelling one would rewrite history, and it
// would restock a listing for a sale that actually happened.
func TestDeletingAnAccountLeavesFinishedOrdersAlone(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	order, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
		ListingID: f.listing, Quantity: 1,
	})
	if err != nil {
		t.Fatalf("ordering: %v", err)
	}
	if _, err := f.db.UpdateOrderStatus(ctx, database.UpdateOrderStatusParams{
		ID: order.ID, Status: "completed",
	}); err != nil {
		t.Fatalf("completing the order: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.seller, "seller"); err != nil {
		t.Fatalf("deleting the seller: %v", err)
	}

	got, err := f.db.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("re-reading the order: %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("status = %q, want completed - a finished sale is a record", got.Status)
	}
}
