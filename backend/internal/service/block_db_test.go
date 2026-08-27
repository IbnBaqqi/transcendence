package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

type blockFixture struct {
	blocks  *BlockService
	chat    *ConversationService
	db      *database.DB
	seller  uuid.UUID
	buyer   uuid.UUID
	listing uuid.UUID
	conv    uuid.UUID
}

func newBlockFixture(t *testing.T) blockFixture {
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
	seller, buyer := mk("seller"), mk("buyer")

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: seller, Title: "Chanterelles", Category: "mushrooms",
		Price: "18.10", Quantity: 10, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating a listing: %v", err)
	}

	chat := NewConversationService(db, notify.Disabled{})

	conv, _, err := chat.StartConversation(ctx, buyer, listing.ID, "are these fresh?")
	if err != nil {
		t.Fatalf("starting the conversation: %v", err)
	}
	// Accepted, so SendMessage tests the block rather than the status.
	if _, err := chat.Accept(ctx, seller, conv.ID); err != nil {
		t.Fatalf("accepting: %v", err)
	}

	return blockFixture{
		blocks: NewBlockService(db.Queries), chat: chat, db: db,
		seller: seller, buyer: buyer, listing: listing.ID, conv: conv.ID,
	}
}

func newListingBy(t *testing.T, db *database.DB, seller uuid.UUID, title string) uuid.UUID {
	t.Helper()

	listing, err := db.CreateListing(context.Background(), database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: seller, Title: title, Category: "berries",
		Price: "9.00", Quantity: 5, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating a listing: %v", err)
	}
	return listing.ID
}

func isForbidden(err error) bool {
	var forbidden *ForbiddenError
	return errors.As(err, &forbidden)
}

func mustFailSend(ctx context.Context, t *testing.T, chat *ConversationService, sender, conv uuid.UUID) error {
	t.Helper()

	_, err := chat.SendMessage(ctx, sender, conv, "hello?")
	if err == nil {
		t.Fatal("expected the send to be refused")
	}
	return err
}

func TestBlockStopsMessagesInBothDirections(t *testing.T) {
	f := newBlockFixture(t)
	ctx := context.Background()

	if err := f.blocks.Block(ctx, f.buyer, f.seller); err != nil {
		t.Fatalf("blocking: %v", err)
	}

	_, err := f.chat.SendMessage(ctx, f.buyer, f.conv, "still there?")
	if !isForbidden(err) {
		t.Errorf("blocker could still send: err = %#v, want *ForbiddenError", err)
	}

	_, err = f.chat.SendMessage(ctx, f.seller, f.conv, "yes, very fresh")
	if !isForbidden(err) {
		t.Errorf("blocked party could still send: err = %#v, want *ForbiddenError", err)
	}
}

func TestBlockStopsANewConversation(t *testing.T) {
	f := newBlockFixture(t)
	ctx := context.Background()

	other := newListingBy(t, f.db, f.seller, "Bilberries")

	if err := f.blocks.Block(ctx, f.seller, f.buyer); err != nil {
		t.Fatalf("blocking: %v", err)
	}

	_, _, err := f.chat.StartConversation(ctx, f.buyer, other, "how much for 5kg?")
	if !isForbidden(err) {
		t.Errorf("err = %#v, want *ForbiddenError", err)
	}
}

func TestUnblockRestoresMessaging(t *testing.T) {
	f := newBlockFixture(t)
	ctx := context.Background()

	if err := f.blocks.Block(ctx, f.buyer, f.seller); err != nil {
		t.Fatalf("blocking: %v", err)
	}
	if err := f.blocks.Unblock(ctx, f.buyer, f.seller); err != nil {
		t.Fatalf("unblocking: %v", err)
	}

	if _, err := f.chat.SendMessage(ctx, f.buyer, f.conv, "sorry, still interested"); err != nil {
		t.Fatalf("messaging after unblock: %v", err)
	}
}

func TestBlockHidesTheThreadFromTheBlockerOnly(t *testing.T) {
	f := newBlockFixture(t)
	ctx := context.Background()

	if err := f.blocks.Block(ctx, f.buyer, f.seller); err != nil {
		t.Fatalf("blocking: %v", err)
	}

	blockerInbox, err := f.chat.ListConversations(ctx, f.buyer)
	if err != nil {
		t.Fatalf("listing the blocker's inbox: %v", err)
	}
	if len(blockerInbox) != 0 {
		t.Errorf("blocker still sees %d threads, want 0", len(blockerInbox))
	}

	blockedInbox, err := f.chat.ListConversations(ctx, f.seller)
	if err != nil {
		t.Fatalf("listing the blocked party's inbox: %v", err)
	}
	if len(blockedInbox) != 1 {
		t.Errorf("blocked party sees %d threads, want 1 - it should go quiet, not vanish", len(blockedInbox))
	}
}

func TestBlockDoesNotChangeAPendingThreadsRefusal(t *testing.T) {
	f := newBlockFixture(t)
	ctx := context.Background()

	other := newListingBy(t, f.db, f.seller, "Lingonberries")
	conv, _, err := f.chat.StartConversation(ctx, f.buyer, other, "hello?")
	if err != nil {
		t.Fatalf("starting the conversation: %v", err)
	}

	before := mustFailSend(ctx, t, f.chat, f.buyer, conv.ID)

	if err := f.blocks.Block(ctx, f.buyer, f.seller); err != nil {
		t.Fatalf("blocking: %v", err)
	}

	after := mustFailSend(ctx, t, f.chat, f.buyer, conv.ID)

	if before.Error() != after.Error() {
		t.Errorf("the refusal changed after blocking: %q then %q - that is observable", before, after)
	}
}

func TestBlockKeepsTheUnreadCountAgreeingWithTheInbox(t *testing.T) {
	f := newBlockFixture(t)
	ctx := context.Background()

	if _, err := f.chat.SendMessage(ctx, f.seller, f.conv, "yes, picked this morning"); err != nil {
		t.Fatalf("seller replying: %v", err)
	}

	unread, err := f.chat.CountUnread(ctx, f.buyer)
	if err != nil {
		t.Fatalf("counting unread: %v", err)
	}
	if unread != 1 {
		t.Fatalf("unread = %d before the block, want 1", unread)
	}

	if err := f.blocks.Block(ctx, f.buyer, f.seller); err != nil {
		t.Fatalf("blocking: %v", err)
	}

	inbox, err := f.chat.ListConversations(ctx, f.buyer)
	if err != nil {
		t.Fatalf("listing the inbox: %v", err)
	}
	unread, err = f.chat.CountUnread(ctx, f.buyer)
	if err != nil {
		t.Fatalf("counting unread: %v", err)
	}

	if len(inbox) != 0 {
		t.Errorf("inbox = %d threads, want 0", len(inbox))
	}
	if unread != 0 {
		t.Errorf("unread = %d, want 0 - the badge counts a thread the inbox hides", unread)
	}
}

func TestBlockDoesNotChangeTheDuplicateThreadRefusal(t *testing.T) {
	f := newBlockFixture(t)
	ctx := context.Background()

	_, _, before := f.chat.StartConversation(ctx, f.buyer, f.listing, "asking again")
	if before == nil {
		t.Fatal("expected the duplicate conversation to be refused")
	}

	if err := f.blocks.Block(ctx, f.seller, f.buyer); err != nil {
		t.Fatalf("blocking: %v", err)
	}

	_, _, after := f.chat.StartConversation(ctx, f.buyer, f.listing, "asking again")
	if after == nil {
		t.Fatal("expected the duplicate conversation to be refused")
	}

	if before.Error() != after.Error() {
		t.Errorf("the refusal changed after blocking: %q then %q - that is observable", before, after)
	}
}

func TestBlockIsIdempotent(t *testing.T) {
	f := newBlockFixture(t)
	ctx := context.Background()

	if err := f.blocks.Block(ctx, f.buyer, f.seller); err != nil {
		t.Fatalf("first block: %v", err)
	}
	if err := f.blocks.Block(ctx, f.buyer, f.seller); err != nil {
		t.Fatalf("second block: %v", err)
	}

	rows, err := f.blocks.List(ctx, f.buyer)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("blocks = %d rows, want 1", len(rows))
	}
}

func TestUnblockingSomeoneNeverBlockedIsNotAnError(t *testing.T) {
	f := newBlockFixture(t)

	if err := f.blocks.Unblock(context.Background(), f.buyer, f.seller); err != nil {
		t.Fatalf("unblocking a non-block: %v", err)
	}
}

func TestBlockingAnUnknownUserIsNotFound(t *testing.T) {
	f := newBlockFixture(t)

	err := f.blocks.Block(context.Background(), f.buyer, uuid.New())

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %#v, want *NotFoundError (a 500 means the FK violation is unhandled)", err)
	}
}

func TestSelfBlockIsRefusedByTheDatabase(t *testing.T) {
	f := newBlockFixture(t)

	err := f.db.BlockUser(context.Background(), database.BlockUserParams{
		BlockerID: f.buyer,
		BlockedID: f.buyer,
	})
	if err == nil {
		t.Fatal("the database allowed a self-block")
	}
}

func TestBlockingDoesNotRemoveAFollow(t *testing.T) {
	f := newBlockFixture(t)
	ctx := context.Background()

	follows := NewFollowService(f.db.Queries)
	if err := follows.Follow(ctx, f.buyer, f.seller); err != nil {
		t.Fatalf("following: %v", err)
	}

	if err := f.blocks.Block(ctx, f.buyer, f.seller); err != nil {
		t.Fatalf("blocking: %v", err)
	}

	following, err := follows.ListFollowing(ctx, f.buyer)
	if err != nil {
		t.Fatalf("listing follows: %v", err)
	}
	if len(following) != 1 {
		t.Errorf("following = %d rows, want 1 - the block deleted it", len(following))
	}
}
