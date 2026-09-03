package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

func notificationsFor(t *testing.T, db *database.DB, userID uuid.UUID) []database.Notification {
	t.Helper()

	rows, err := db.ListNotifications(context.Background(), database.ListNotificationsParams{
		UserID: userID,
		Limit:  30,
	})
	if err != nil {
		t.Fatalf("reading notifications: %v", err)
	}
	return rows
}

func kindsOf(rows []database.Notification) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Kind)
	}
	return out
}

// Placing an order tells the seller, and nobody else - a buyer who already
// knows what they just did does not need telling.
func TestPlacingAnOrderRecordsANotificationForTheSeller(t *testing.T) {
	f := newOrderFixture(t)

	seller := notificationsFor(t, f.db, f.seller)
	if got := kindsOf(seller); len(got) != 1 || got[0] != notifyKindOrderPlaced {
		t.Errorf("seller notifications = %v, want [order_placed]", got)
	}

	if got := notificationsFor(t, f.db, f.buyer); len(got) != 0 {
		t.Errorf("buyer notifications = %v, want none", kindsOf(got))
	}

	if !seller[0].OrderID.Valid || seller[0].OrderID.UUID != f.order.ID {
		t.Error("the notification does not link to the order it is about")
	}
	if seller[0].ListingTitle != f.order.ListingTitle {
		t.Errorf("listing_title = %q, want the order's %q", seller[0].ListingTitle, f.order.ListingTitle)
	}
	if seller[0].ReadAt.Valid {
		t.Error("a new notification is already read")
	}
}

// Cancelling tells the other party, whichever one that is.
func TestCancellingRecordsANotificationForTheOtherParty(t *testing.T) {
	f := newOrderFixture(t)
	ctx := context.Background()

	if _, err := f.svc.CancelOrder(ctx, f.buyer, f.order.ID); err != nil {
		t.Fatalf("cancelling: %v", err)
	}

	if got := kindsOf(notificationsFor(t, f.db, f.seller)); len(got) != 2 || got[0] != notifyKindOrderCancelled {
		t.Errorf("seller notifications = %v, want the cancellation newest", got)
	}
	// The buyer cancelled, so the buyer is not told about it.
	if got := notificationsFor(t, f.db, f.buyer); len(got) != 0 {
		t.Errorf("buyer notifications = %v, want none", kindsOf(got))
	}
}

// A handover that completes the order announces nothing: the completion is the
// news, not the handover. The sequence matters - the guard only fires when the
// handover is the second mark, so the buyer has to confirm receipt first.
func TestAHandoverThatCompletesTheOrderAnnouncesNothing(t *testing.T) {
	f := newOrderFixture(t)
	ctx := context.Background()

	if _, err := f.svc.ConfirmOrder(ctx, f.seller, f.order.ID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	// The buyer marks first, so the seller's handover is what completes it.
	if _, err := f.svc.ReceiveOrder(ctx, f.buyer, f.order.ID); err != nil {
		t.Fatalf("receive: %v", err)
	}

	before := kindsOf(notificationsFor(t, f.db, f.buyer))

	if _, err := f.svc.HandoverOrder(ctx, f.seller, f.order.ID); err != nil {
		t.Fatalf("handover: %v", err)
	}

	if got := kindsOf(notificationsFor(t, f.db, f.buyer)); len(got) != len(before) {
		t.Errorf("buyer notifications went from %v to %v; the completing handover should add nothing", before, got)
	}
}

// The ordinary case, for contrast: a handover that leaves the order confirmed
// does tell the buyer.
func TestAHandoverBeforeReceiptTellsTheBuyer(t *testing.T) {
	f := newOrderFixture(t)
	ctx := context.Background()

	if _, err := f.svc.ConfirmOrder(ctx, f.seller, f.order.ID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if _, err := f.svc.HandoverOrder(ctx, f.seller, f.order.ID); err != nil {
		t.Fatalf("handover: %v", err)
	}

	if got := kindsOf(notificationsFor(t, f.db, f.buyer)); len(got) != 1 || got[0] != notifyKindOrderHandedOver {
		t.Errorf("buyer notifications = %v, want [order_handed_over]", got)
	}
}

// The row commits with the change it describes. A refused action leaves no
// trace of a thing that never happened.
func TestARefusedActionLeavesNoNotification(t *testing.T) {
	f := newOrderFixture(t)
	ctx := context.Background()

	before := len(notificationsFor(t, f.db, f.seller))

	// The buyer cannot confirm - only the seller can.
	if _, err := f.svc.ConfirmOrder(ctx, f.buyer, f.order.ID); err == nil {
		t.Fatal("the buyer was allowed to confirm")
	}

	if after := len(notificationsFor(t, f.db, f.seller)); after != before {
		t.Errorf("notifications went from %d to %d for an action that was refused", before, after)
	}
}

// Mark-all-read is scoped to the caller: one user clearing their list must not
// clear anybody else's.
func TestMarkAllReadTouchesOnlyTheCaller(t *testing.T) {
	f := newOrderFixture(t)
	ctx := context.Background()

	// Give the buyer one too, by having the seller cancel.
	if _, err := f.svc.CancelOrder(ctx, f.seller, f.order.ID); err != nil {
		t.Fatalf("cancelling: %v", err)
	}

	svc := NewNotificationService(f.db.Queries)
	cleared, err := svc.MarkAllRead(ctx, f.seller)
	if err != nil {
		t.Fatalf("marking read: %v", err)
	}
	if cleared == 0 {
		t.Fatal("nothing was marked read")
	}

	for _, row := range notificationsFor(t, f.db, f.seller) {
		if !row.ReadAt.Valid {
			t.Errorf("the caller still has an unread %s", row.Kind)
		}
	}
	for _, row := range notificationsFor(t, f.db, f.buyer) {
		if row.ReadAt.Valid {
			t.Errorf("the other user's %s was marked read too", row.Kind)
		}
	}

	// Idempotent: the second call matches nothing.
	again, err := svc.MarkAllRead(ctx, f.seller)
	if err != nil {
		t.Fatalf("marking read again: %v", err)
	}
	if again != 0 {
		t.Errorf("a second mark-all-read cleared %d rows, want 0", again)
	}
}
