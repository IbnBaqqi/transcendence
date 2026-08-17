package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

type orderFixture struct {
	svc    *OrderService
	db     *database.DB
	seller uuid.UUID
	buyer  uuid.UUID
	order  database.Order
}

func newOrderFixture(t *testing.T) orderFixture {
	t.Helper()

	db := testdb.New(t)
	ctx := context.Background()

	mk := func(name string) uuid.UUID {
		user, err := db.CreateUser(ctx, database.CreateUserParams{
			Username: name, Email: name + "@example.test", Password: "irrelevant",
		})
		if err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		return user.ID
	}
	seller, buyer := mk("seller"), mk("buyer")

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		SellerID: seller, Title: "Chanterelles", Category: "mushrooms",
		Price: "18.10", Quantity: 10, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating a listing: %v", err)
	}

	svc := NewOrderService(db)
	order, err := svc.CreateOrder(ctx, buyer, dtos.CreateOrderInput{ListingID: listing.ID, Quantity: 2})
	if err != nil {
		t.Fatalf("placing the order: %v", err)
	}

	return orderFixture{svc: svc, db: db, seller: seller, buyer: buyer, order: order}
}

func (f orderFixture) events(t *testing.T) []database.OrderEvent {
	t.Helper()

	rows, err := f.db.ListOrderEvents(context.Background(), f.order.ID)
	if err != nil {
		t.Fatalf("reading the timeline: %v", err)
	}
	return rows
}

func TestTimelineRecordsTheWholeLifecycle(t *testing.T) {
	f := newOrderFixture(t)
	ctx := context.Background()

	if _, err := f.svc.ConfirmOrder(ctx, f.seller, f.order.ID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if _, err := f.svc.HandoverOrder(ctx, f.seller, f.order.ID); err != nil {
		t.Fatalf("handover: %v", err)
	}
	if _, err := f.svc.ReceiveOrder(ctx, f.buyer, f.order.ID); err != nil {
		t.Fatalf("receive: %v", err)
	}

	want := []struct {
		from  string
		to    string
		note  string
		actor uuid.UUID
	}{
		{"", "pending", "", f.buyer},
		{"pending", "confirmed", "", f.seller},
		{"confirmed", "confirmed", "seller marked handover", f.seller},
		{"confirmed", "confirmed", "buyer confirmed receipt", f.buyer},
		{"confirmed", "completed", "", f.buyer},
	}

	got := f.events(t)
	if len(got) != len(want) {
		t.Fatalf("events = %d, want %d: %+v", len(got), len(want), got)
	}

	for i, w := range want {
		e := got[i]
		if e.FromStatus.String != w.from || e.FromStatus.Valid != (w.from != "") {
			t.Errorf("event %d from = %v, want %q", i, e.FromStatus, w.from)
		}
		if e.ToStatus != w.to {
			t.Errorf("event %d to = %q, want %q", i, e.ToStatus, w.to)
		}
		if e.Note.String != w.note {
			t.Errorf("event %d note = %q, want %q", i, e.Note.String, w.note)
		}
		if e.ActorID.UUID != w.actor {
			t.Errorf("event %d actor = %v, want %v", i, e.ActorID.UUID, w.actor)
		}
	}
}

func TestRefusedTransitionRecordsNothing(t *testing.T) {
	f := newOrderFixture(t)
	ctx := context.Background()

	before := len(f.events(t))

	if _, err := f.svc.ReceiveOrder(ctx, f.buyer, f.order.ID); err == nil {
		t.Fatal("receiving a pending order should have failed")
	}
	if _, err := f.svc.ConfirmOrder(ctx, f.buyer, f.order.ID); err == nil {
		t.Fatal("a buyer confirming should have failed")
	}
	if _, err := f.svc.HandoverOrder(ctx, f.seller, f.order.ID); err == nil {
		t.Fatal("handing over a pending order should have failed")
	}

	if after := len(f.events(t)); after != before {
		t.Errorf("events = %d, want %d - a refused transition was recorded", after, before)
	}
}

func TestTransitionRollsBackWhenTheEventFails(t *testing.T) {
	f := newOrderFixture(t)
	ctx := context.Background()

	if _, err := f.db.Exec("ALTER TABLE order_events RENAME TO order_events_hidden"); err != nil {
		t.Fatalf("hiding the events table: %v", err)
	}

	if _, err := f.svc.ConfirmOrder(ctx, f.seller, f.order.ID); err == nil {
		t.Fatal("confirm succeeded despite the event write failing")
	}

	order, err := f.db.GetOrder(ctx, f.order.ID)
	if err != nil {
		t.Fatalf("re-reading the order: %v", err)
	}
	if order.Status != "pending" {
		t.Errorf("status = %q, want pending - the transition outlived its failed event", order.Status)
	}
}

func TestCancelRecordsWhoDidIt(t *testing.T) {
	f := newOrderFixture(t)

	if _, err := f.svc.CancelOrder(context.Background(), f.seller, f.order.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	got := f.events(t)
	last := got[len(got)-1]
	if last.ToStatus != "cancelled" {
		t.Fatalf("last event = %q, want cancelled", last.ToStatus)
	}
	if last.ActorID.UUID != f.seller {
		t.Errorf("actor = %v, want the seller %v", last.ActorID.UUID, f.seller)
	}
}

// Migration 016. Nothing else asserts this, and the first version of that
// migration silently dropped the buyer's foreign key entirely - every other
// test stayed green while a seller delete wiped the order and its history.
func TestAnOrdersPartiesCannotBeDeleted(t *testing.T) {
	tests := []struct {
		name  string
		party func(orderFixture) uuid.UUID
	}{
		{"the buyer", func(f orderFixture) uuid.UUID { return f.buyer }},
		{"the seller", func(f orderFixture) uuid.UUID { return f.seller }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newOrderFixture(t)

			if _, err := f.db.Exec("DELETE FROM users WHERE id = $1", tt.party(f)); err == nil {
				t.Error("the delete was allowed - order history can be erased")
			}

			var orders, events int
			if err := f.db.QueryRow("SELECT count(*) FROM orders").Scan(&orders); err != nil {
				t.Fatal(err)
			}
			if err := f.db.QueryRow("SELECT count(*) FROM order_events").Scan(&events); err != nil {
				t.Fatal(err)
			}
			if orders != 1 || events != 1 {
				t.Errorf("orders = %d, events = %d, want 1 and 1", orders, events)
			}
		})
	}
}

// A JWT outlives the account it names, and Authenticate does no database
// lookup - so a deleted user still reaches the service. That must be a 404,
// not the driver error a foreign key violation would otherwise produce.
func TestCreateOrderByADeletedUserIsNotFound(t *testing.T) {
	f := newOrderFixture(t)
	ctx := context.Background()

	listing, err := f.db.CreateListing(ctx, database.CreateListingParams{
		SellerID: f.seller, Title: "More chanterelles", Category: "mushrooms",
		Price: "12.00", Quantity: 5, Unit: "kg",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Deletable because they have no orders yet - the RESTRICT guard only
	// covers the parties to an existing order.
	ghost, err := f.db.CreateUser(ctx, database.CreateUserParams{
		Username: "ghost", Email: "ghost@example.test", Password: "irrelevant",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec("DELETE FROM users WHERE id = $1", ghost.ID); err != nil {
		t.Fatal(err)
	}

	_, err = f.svc.CreateOrder(ctx, ghost.ID, dtos.CreateOrderInput{ListingID: listing.ID, Quantity: 1})

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %#v, want *NotFoundError (a raw pq error here is a 500)", err)
	}
}

// The guard is on the parties, not on users in general.
func TestAnUninvolvedUserCanStillBeDeleted(t *testing.T) {
	f := newOrderFixture(t)

	bystander, err := f.db.CreateUser(context.Background(), database.CreateUserParams{
		Username: "bystander", Email: "bystander@example.test", Password: "irrelevant",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.db.Exec("DELETE FROM users WHERE id = $1", bystander.ID); err != nil {
		t.Errorf("deleting an uninvolved user was refused: %v", err)
	}
}

func TestListEventsIsForParticipantsOnly(t *testing.T) {
	f := newOrderFixture(t)
	ctx := context.Background()

	stranger, err := f.db.CreateUser(ctx, database.CreateUserParams{
		Username: "stranger", Email: "stranger@example.test", Password: "irrelevant",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.ListEvents(ctx, f.buyer, f.order.ID); err != nil {
		t.Errorf("the buyer was refused their own order: %v", err)
	}
	if _, err := f.svc.ListEvents(ctx, f.seller, f.order.ID); err != nil {
		t.Errorf("the seller was refused: %v", err)
	}

	var forbidden *ForbiddenError
	if _, err := f.svc.ListEvents(ctx, stranger.ID, f.order.ID); !errors.As(err, &forbidden) {
		t.Errorf("stranger err = %v, want *ForbiddenError", err)
	}

	var notFound *NotFoundError
	if _, err := f.svc.ListEvents(ctx, f.buyer, 999999); !errors.As(err, &notFound) {
		t.Errorf("unknown order err = %v, want *NotFoundError", err)
	}
}
