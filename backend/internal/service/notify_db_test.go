package service

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

type recorder struct {
	mu   sync.Mutex
	sent []notify.Message
}

func (r *recorder) Notify(_ context.Context, m notify.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, m)
}

func (r *recorder) Close() {}

func (r *recorder) messages() []notify.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]notify.Message(nil), r.sent...)
}

func (r *recorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = nil
}

func (r *recorder) only(t *testing.T, kind notify.Kind, to string) {
	t.Helper()

	msgs := r.messages()
	if len(msgs) != 1 {
		t.Fatalf("sent %d notifications, want 1: %+v", len(msgs), msgs)
	}
	if msgs[0].Kind != kind {
		t.Errorf("kind = %q, want %q", msgs[0].Kind, kind)
	}
	if msgs[0].To != to {
		t.Errorf("to = %q, want %q", msgs[0].To, to)
	}
}

type notifyFixture struct {
	orders      *OrderService
	chat        *ConversationService
	db          *database.DB
	rec         *recorder
	seller      uuid.UUID
	buyer       uuid.UUID
	sellerEmail string
	buyerEmail  string
	listing     uuid.UUID
}

func newNotifyFixture(t *testing.T) notifyFixture {
	t.Helper()

	db := testdb.New(t)
	ctx := context.Background()
	rec := &recorder{}

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

	return notifyFixture{
		orders:      NewOrderService(db, rec),
		chat:        NewConversationService(db, rec),
		db:          db,
		rec:         rec,
		seller:      seller,
		buyer:       buyer,
		sellerEmail: "seller@example.test",
		buyerEmail:  "buyer@example.test",
		listing:     listing.ID,
	}
}

func (f notifyFixture) placeOrder(t *testing.T) database.Order {
	t.Helper()

	order, err := f.orders.CreateOrder(context.Background(), f.buyer,
		dtos.CreateOrderInput{ListingID: f.listing, Quantity: 2})
	if err != nil {
		t.Fatalf("placing the order: %v", err)
	}
	return order
}

func TestPlacingAnOrderNotifiesTheSeller(t *testing.T) {
	f := newNotifyFixture(t)
	f.placeOrder(t)
	f.rec.only(t, notify.KindOrderPlaced, f.sellerEmail)
}

func TestHandoverNotifiesTheBuyerBeforeTheOrderCompletes(t *testing.T) {
	f := newNotifyFixture(t)
	ctx := context.Background()

	order := f.placeOrder(t)
	if _, err := f.orders.ConfirmOrder(ctx, f.seller, order.ID); err != nil {
		t.Fatalf("confirming: %v", err)
	}
	f.rec.reset()

	handed, err := f.orders.HandoverOrder(ctx, f.seller, order.ID)
	if err != nil {
		t.Fatalf("handing over: %v", err)
	}
	if handed.Status == "completed" {
		t.Fatal("the order completed, so this is not exercising the early-return path")
	}

	f.rec.only(t, notify.KindOrderHandedOver, f.buyerEmail)
}

func TestHandoverAfterReceiveDoesNotAnnounceAShipment(t *testing.T) {
	f := newNotifyFixture(t)
	ctx := context.Background()

	order := f.placeOrder(t)
	if _, err := f.orders.ConfirmOrder(ctx, f.seller, order.ID); err != nil {
		t.Fatalf("confirming: %v", err)
	}
	if _, err := f.orders.ReceiveOrder(ctx, f.buyer, order.ID); err != nil {
		t.Fatalf("receiving: %v", err)
	}
	f.rec.reset()

	done, err := f.orders.HandoverOrder(ctx, f.seller, order.ID)
	if err != nil {
		t.Fatalf("handing over: %v", err)
	}
	if done.Status != "completed" {
		t.Fatalf("status = %q, want completed - this is not exercising the second commit point", done.Status)
	}

	if msgs := f.rec.messages(); len(msgs) != 0 {
		t.Errorf("sent %d notifications for an order that was already received: %+v", len(msgs), msgs)
	}
}

func TestADisconnectedClientStillNotifies(t *testing.T) {
	f := newNotifyFixture(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	notifyUser(ctx, f.db.Queries, f.rec, f.seller,
		func(email, _ string) notify.Message {
			return notify.OrderPlaced(email, "Chanterelles", 2, "kg")
		})

	f.rec.only(t, notify.KindOrderPlaced, f.sellerEmail)
}

func TestCancellingNotifiesTheOtherParty(t *testing.T) {
	t.Run("buyer cancels, seller hears", func(t *testing.T) {
		f := newNotifyFixture(t)
		order := f.placeOrder(t)
		f.rec.reset()

		if _, err := f.orders.CancelOrder(context.Background(), f.buyer, order.ID); err != nil {
			t.Fatalf("cancelling: %v", err)
		}
		f.rec.only(t, notify.KindOrderCancelled, f.sellerEmail)
	})

	t.Run("seller cancels, buyer hears", func(t *testing.T) {
		f := newNotifyFixture(t)
		order := f.placeOrder(t)
		f.rec.reset()

		if _, err := f.orders.CancelOrder(context.Background(), f.seller, order.ID); err != nil {
			t.Fatalf("cancelling: %v", err)
		}
		f.rec.only(t, notify.KindOrderCancelled, f.buyerEmail)
	})
}

func TestConfirmAndReceiveNotifyNobody(t *testing.T) {
	f := newNotifyFixture(t)
	ctx := context.Background()

	order := f.placeOrder(t)
	f.rec.reset()

	if _, err := f.orders.ConfirmOrder(ctx, f.seller, order.ID); err != nil {
		t.Fatalf("confirming: %v", err)
	}
	if _, err := f.orders.ReceiveOrder(ctx, f.buyer, order.ID); err != nil {
		t.Fatalf("receiving: %v", err)
	}

	if msgs := f.rec.messages(); len(msgs) != 0 {
		t.Errorf("sent %d notifications, want 0: %+v", len(msgs), msgs)
	}
}

func TestStartingAChatNotifiesTheSeller(t *testing.T) {
	f := newNotifyFixture(t)

	if _, _, err := f.chat.StartConversation(context.Background(), f.buyer, f.listing, "fresh?"); err != nil {
		t.Fatalf("starting the conversation: %v", err)
	}
	f.rec.only(t, notify.KindChatRequest, f.sellerEmail)
}

func TestAFailedActionNotifiesNobody(t *testing.T) {
	f := newNotifyFixture(t)

	_, err := f.orders.CreateOrder(context.Background(), f.buyer,
		dtos.CreateOrderInput{ListingID: f.listing, Quantity: 999})
	if err == nil {
		t.Fatal("expected the order to be refused for stock")
	}

	if msgs := f.rec.messages(); len(msgs) != 0 {
		t.Errorf("sent %d notifications for a refused order: %+v", len(msgs), msgs)
	}
}

func TestTheCancellationEmailReadsForBothSides(t *testing.T) {
	f := newNotifyFixture(t)
	ctx := context.Background()

	order := f.placeOrder(t)
	f.rec.reset()

	if _, err := f.orders.CancelOrder(ctx, f.seller, order.ID); err != nil {
		t.Fatalf("cancelling: %v", err)
	}

	msgs := f.rec.messages()
	if len(msgs) != 1 {
		t.Fatalf("sent %d notifications, want 1", len(msgs))
	}

	// The seller cancelled, so this reaches the buyer. Seller-facing detail
	// ("the stock released") is nonsense to them, and there is no real money
	// in this flow to refund.
	for _, phrase := range []string{"stock", "refund"} {
		if strings.Contains(strings.ToLower(msgs[0].Body), phrase) {
			t.Errorf("the buyer's cancellation email mentions %q:\n%s", phrase, msgs[0].Body)
		}
	}
}
