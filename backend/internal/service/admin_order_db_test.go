package service

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/storage"
)

type adminOrderFixture struct {
	orderFixture
	admins *AdminOrderService
	admin  uuid.UUID
}

func newAdminOrderFixture(t *testing.T) adminOrderFixture {
	t.Helper()

	f := newOrderFixture(t)

	admin, err := f.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:       database.NewID(),
		Username: "admin", Email: "admin@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating the admin: %v", err)
	}

	return adminOrderFixture{
		orderFixture: f,
		admins:       NewAdminOrderService(f.db, notify.Disabled{}),
		admin:        admin.ID,
	}
}

func (f adminOrderFixture) stick(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	if _, err := f.svc.ConfirmOrder(ctx, f.seller, f.order.ID); err != nil {
		t.Fatalf("confirming: %v", err)
	}
	if _, err := f.svc.HandoverOrder(ctx, f.seller, f.order.ID); err != nil {
		t.Fatalf("handing over: %v", err)
	}

	if _, err := f.db.ExecContext(ctx,
		`UPDATE orders SET seller_handed_over_at = now() - interval '8 days' WHERE id = $1`,
		f.order.ID); err != nil {
		t.Fatalf("backdating the handover: %v", err)
	}

	resolvable, err := f.db.GetOrderResolvability(ctx, f.order.ID)
	if err != nil {
		t.Fatalf("re-reading: %v", err)
	}
	if !resolvable.Stuck || resolvable.Stranded {
		t.Fatalf("the fixture did not produce a handshake-stuck order: stuck=%v stranded=%v",
			resolvable.Stuck, resolvable.Stranded)
	}
}

func TestOnlyAStuckOrderCanBeResolved(t *testing.T) {
	ctx := context.Background()

	t.Run("a pending order is refused", func(t *testing.T) {
		f := newAdminOrderFixture(t)

		var conflict *ConflictError
		if _, err := f.admins.Resolve(ctx, f.admin, f.order.ID, "refunded", "why"); !errors.As(err, &conflict) {
			t.Errorf("err = %#v, want *ConflictError", err)
		}
	})

	t.Run("a confirmed order with no handshake is refused", func(t *testing.T) {
		f := newAdminOrderFixture(t)
		if _, err := f.svc.ConfirmOrder(ctx, f.seller, f.order.ID); err != nil {
			t.Fatalf("confirming: %v", err)
		}

		var conflict *ConflictError
		if _, err := f.admins.Resolve(ctx, f.admin, f.order.ID, "refunded", "why"); !errors.As(err, &conflict) {
			t.Errorf("err = %#v, want *ConflictError - both marks are unset, so it is not stuck", err)
		}
	})

	t.Run("a stuck order is resolved", func(t *testing.T) {
		f := newAdminOrderFixture(t)
		f.stick(t)

		order, err := f.admins.Resolve(ctx, f.admin, f.order.ID, "refunded", "buyer never confirmed")
		if err != nil {
			t.Fatalf("resolving: %v", err)
		}
		if order.Status != "refunded" {
			t.Errorf("status = %q, want refunded", order.Status)
		}
	})

	t.Run("resolving twice is refused", func(t *testing.T) {
		f := newAdminOrderFixture(t)
		f.stick(t)

		if _, err := f.admins.Resolve(ctx, f.admin, f.order.ID, "refunded", "first"); err != nil {
			t.Fatalf("resolving: %v", err)
		}

		var conflict *ConflictError
		if _, err := f.admins.Resolve(ctx, f.admin, f.order.ID, "cancelled", "second"); !errors.As(err, &conflict) {
			t.Errorf("err = %#v, want *ConflictError", err)
		}
	})
}

func TestOnlyANonTradeReturnsTheStock(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		outcome string
		want    int32
	}{
		{"cancelled", 10},
		{"refunded", 10},
		{"completed", 8},
	}

	for _, tt := range tests {
		t.Run(tt.outcome, func(t *testing.T) {
			f := newAdminOrderFixture(t)
			f.stick(t)

			before, err := f.db.GetListing(ctx, f.order.ListingID)
			if err != nil {
				t.Fatalf("reading the listing: %v", err)
			}
			if before.Quantity != 8 {
				t.Fatalf("the fixture should have 8 left of 10, got %d", before.Quantity)
			}

			if _, err := f.admins.Resolve(ctx, f.admin, f.order.ID, tt.outcome, "why"); err != nil {
				t.Fatalf("resolving: %v", err)
			}

			after, err := f.db.GetListing(ctx, f.order.ListingID)
			if err != nil {
				t.Fatalf("re-reading the listing: %v", err)
			}
			if after.Quantity != tt.want {
				t.Errorf("quantity = %d, want %d", after.Quantity, tt.want)
			}
		})
	}
}

func TestAResolutionSaysWhoWhatAndWhy(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()
	f.stick(t)

	if _, err := f.admins.Resolve(ctx, f.admin, f.order.ID, "refunded", "  buyer never confirmed  "); err != nil {
		t.Fatalf("resolving: %v", err)
	}

	events, err := f.db.ListOrderEvents(ctx, f.order.ID)
	if err != nil {
		t.Fatalf("reading the events: %v", err)
	}

	last := events[len(events)-1]
	if last.FromStatus.String != "confirmed" || last.ToStatus != "refunded" {
		t.Errorf("event = %s -> %s, want confirmed -> refunded", last.FromStatus.String, last.ToStatus)
	}
	if last.Note.String != "buyer never confirmed" {
		t.Errorf("note = %q, want the trimmed reason", last.Note.String)
	}
	if last.ActorID.UUID != f.admin {
		t.Error("the event does not name the admin who resolved it")
	}
}

func TestAResolutionNeedsAKnownOutcomeAndAReason(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		outcome string
		reason  string
	}{
		{"an unknown outcome", "vanished", "why"},
		{"an empty outcome", "", "why"},
		{"no reason", "refunded", ""},
		{"a reason of only spaces", "refunded", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdminOrderFixture(t)
			f.stick(t)

			var invalid *ValidationError
			if _, err := f.admins.Resolve(ctx, f.admin, f.order.ID, tt.outcome, tt.reason); !errors.As(err, &invalid) {
				t.Errorf("err = %#v, want *ValidationError", err)
			}
		})
	}
}

func TestTheStuckFilterMatchesTheGoPredicate(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()
	f.stick(t)

	page, err := f.admins.List(ctx, dtos.AdminOrderQuery{Stuck: "true"})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("stuck=true returned %d items, total %d, want 1 and 1", len(page.Items), page.Total)
	}
	if !page.Items[0].Stuck {
		t.Error("the row the stuck filter selected is not flagged stuck on the response")
	}

	notStuck, err := f.admins.List(ctx, dtos.AdminOrderQuery{Stuck: "false"})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if notStuck.Total != 0 {
		t.Errorf("stuck=false returned %d, want 0 - the only order is stuck", notStuck.Total)
	}
}

func TestAdminOrderFiltersAgreeOnRowsAndTotal(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()
	f.stick(t)

	tests := []struct {
		name  string
		query dtos.AdminOrderQuery
		want  int
	}{
		{"no filter", dtos.AdminOrderQuery{}, 1},
		{"matching status", dtos.AdminOrderQuery{Status: "confirmed"}, 1},
		{"other status", dtos.AdminOrderQuery{Status: "completed"}, 0},
		{"stuck", dtos.AdminOrderQuery{Stuck: "true"}, 1},
		{"a range that covers it", dtos.AdminOrderQuery{CreatedFrom: "2000-01-01"}, 1},
		{"a range that ends before it", dtos.AdminOrderQuery{CreatedTo: "2000-01-01"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := f.admins.List(ctx, tt.query)
			if err != nil {
				t.Fatalf("listing: %v", err)
			}
			if len(page.Items) != tt.want {
				t.Errorf("items = %d, want %d", len(page.Items), tt.want)
			}
			if int(page.Total) != tt.want {
				t.Errorf("total = %d, want %d - the count query disagrees with the rows", page.Total, tt.want)
			}
		})
	}
}

func TestAdminOrderListRejectsBadInput(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()

	tests := []struct {
		name  string
		query dtos.AdminOrderQuery
	}{
		{"unknown status", dtos.AdminOrderQuery{Status: "vanished"}},
		{"unparseable stuck", dtos.AdminOrderQuery{Stuck: "maybe"}},
		{"unparseable date", dtos.AdminOrderQuery{CreatedFrom: "yesterday"}},
		{"a range that ends before it starts", dtos.AdminOrderQuery{CreatedFrom: "2026-09-01", CreatedTo: "2026-08-01"}},
		{"page zero", dtos.AdminOrderQuery{Page: "0"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var invalid *ValidationError
			if _, err := f.admins.List(ctx, tt.query); !errors.As(err, &invalid) {
				t.Errorf("err = %#v, want *ValidationError", err)
			}
		})
	}
}

func TestAFreshHandoverIsNotYetStuck(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()

	if _, err := f.svc.ConfirmOrder(ctx, f.seller, f.order.ID); err != nil {
		t.Fatalf("confirming: %v", err)
	}
	if _, err := f.svc.HandoverOrder(ctx, f.seller, f.order.ID); err != nil {
		t.Fatalf("handing over: %v", err)
	}

	page, err := f.admins.List(ctx, dtos.AdminOrderQuery{Stuck: "true"})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if page.Total != 0 {
		t.Errorf("a handover seconds old is already in the queue: total = %d, want 0", page.Total)
	}

	var conflict *ConflictError
	if _, err := f.admins.Resolve(ctx, f.admin, f.order.ID, "cancelled", "too soon"); !errors.As(err, &conflict) {
		t.Errorf("resolving a live handover: err = %#v, want *ConflictError - the buyer can still receive it", err)
	}
}

func TestABuyerOnlyHandshakeAlsoGetsStuck(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()

	if _, err := f.svc.ConfirmOrder(ctx, f.seller, f.order.ID); err != nil {
		t.Fatalf("confirming: %v", err)
	}
	if _, err := f.svc.ReceiveOrder(ctx, f.buyer, f.order.ID); err != nil {
		t.Fatalf("receiving: %v", err)
	}
	if _, err := f.db.ExecContext(ctx,
		`UPDATE orders SET buyer_received_at = now() - interval '8 days' WHERE id = $1`,
		f.order.ID); err != nil {
		t.Fatalf("backdating: %v", err)
	}

	page, err := f.admins.List(ctx, dtos.AdminOrderQuery{Stuck: "true"})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if page.Total != 1 || !page.Items[0].Stuck {
		t.Errorf("the buyer-only arm of the XOR is not treated as stuck: total = %d", page.Total)
	}
}

func TestTheDateFiltersAreHalfOpenAtTheBoundary(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()

	order, err := f.db.GetOrder(ctx, f.order.ID)
	if err != nil {
		t.Fatalf("reading the order: %v", err)
	}
	at := order.CreatedAt.Time.UTC().Format(time.RFC3339Nano)

	from, err := f.admins.List(ctx, dtos.AdminOrderQuery{CreatedFrom: at})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if from.Total != 1 {
		t.Errorf("created_from at the exact timestamp excluded it: total = %d, want 1 (inclusive)", from.Total)
	}

	to, err := f.admins.List(ctx, dtos.AdminOrderQuery{CreatedTo: at})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if to.Total != 0 {
		t.Errorf("created_to at the exact timestamp included it: total = %d, want 0 (exclusive)", to.Total)
	}
}

func TestTheListEchoesItsPagingAndCountsPages(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()

	page, err := f.admins.List(ctx, dtos.AdminOrderQuery{Page: "2", Limit: "1"})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if page.Page != 2 || page.Limit != 1 {
		t.Errorf("paging echo = page %d limit %d, want 2 and 1", page.Page, page.Limit)
	}
	if page.TotalPages != 1 {
		t.Errorf("total_pages = %d, want 1 for one order at a limit of 1", page.TotalPages)
	}
	if page.Items == nil {
		t.Error("an empty page returned a nil slice, which marshals to null")
	}
}

func TestTheLimitIsClampedRatherThanRejected(t *testing.T) {
	f := newAdminOrderFixture(t)

	page, err := f.admins.List(context.Background(), dtos.AdminOrderQuery{Limit: "5000"})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if page.Limit != maxLimit {
		t.Errorf("limit = %d, want it clamped to %d", page.Limit, maxLimit)
	}
}

func TestAnOverlongReasonIsRejected(t *testing.T) {
	f := newAdminOrderFixture(t)
	f.stick(t)

	long := strings.Repeat("a", maxResolutionReason+1)

	var invalid *ValidationError
	if _, err := f.admins.Resolve(context.Background(), f.admin, f.order.ID, "refunded", long); !errors.As(err, &invalid) {
		t.Errorf("err = %#v, want *ValidationError", err)
	}
}

func TestAReasonIsStrippedOfControlCharacters(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()
	f.stick(t)

	if _, err := f.admins.Resolve(ctx, f.admin, f.order.ID, "refunded", "buyer\u202evanished"); err != nil {
		t.Fatalf("resolving: %v", err)
	}

	events, err := f.db.ListOrderEvents(ctx, f.order.ID)
	if err != nil {
		t.Fatalf("reading the events: %v", err)
	}

	note := events[len(events)-1].Note.String
	if strings.ContainsRune(note, '\u202e') {
		t.Errorf("a bidi override survived into an admin-visible note: %q", note)
	}
}

func (f adminOrderFixture) deleteAccount(t *testing.T, id uuid.UUID, username string) {
	t.Helper()
	ctx := context.Background()

	if err := f.db.EnsureProfile(ctx, id); err != nil {
		t.Fatalf("creating %s's profile: %v", username, err)
	}

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	if err := NewUserService(f.db, files).DeleteAccount(ctx, id, username); err != nil {
		t.Fatalf("deleting %s: %v", username, err)
	}
}

func TestAnOrderBothPartiesAbandonedIsStuckImmediately(t *testing.T) {
	// Both from-states of actionCancel, because the stranded arm names them
	// explicitly: covering only "pending" lets the arm narrow to it unnoticed.
	for _, status := range []string{"pending", "confirmed"} {
		t.Run("an order left "+status, func(t *testing.T) {
			f := newAdminOrderFixture(t)
			ctx := context.Background()

			if status == "confirmed" {
				if _, err := f.svc.ConfirmOrder(ctx, f.seller, f.order.ID); err != nil {
					t.Fatalf("confirming: %v", err)
				}
			}

			f.deleteAccount(t, f.buyer, "buyer")
			f.deleteAccount(t, f.seller, "seller")

			order, err := f.db.GetOrder(ctx, f.order.ID)
			if err != nil {
				t.Fatalf("re-reading: %v", err)
			}
			if order.Status != status {
				t.Fatalf("the fixture produced %q, not %q", order.Status, status)
			}

			resolvable, err := f.db.GetOrderResolvability(ctx, f.order.ID)
			if err != nil {
				t.Fatalf("reading resolvability: %v", err)
			}
			if !resolvable.Stranded {
				t.Error("an order neither party can act on is not flagged stranded")
			}
			if !resolvable.Stuck {
				t.Error("stranded did not make the order stuck")
			}

			page, err := f.admins.List(ctx, dtos.AdminOrderQuery{Stuck: "true"})
			if err != nil {
				t.Fatalf("listing: %v", err)
			}
			if page.Total != 1 || len(page.Items) != 1 {
				t.Fatalf("stuck=true returned %d items, total %d, want 1 and 1 - no seven-day wait applies here",
					len(page.Items), page.Total)
			}
			if !page.Items[0].Stuck {
				t.Error("the listed row is not flagged stuck on the response")
			}
		})
	}
}

func TestAStrandedOrderResolvesToCancelledAndReturnsStock(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()

	f.deleteAccount(t, f.buyer, "buyer")
	f.deleteAccount(t, f.seller, "seller")

	resolved, err := f.admins.Resolve(ctx, f.admin, f.order.ID, "cancelled", "both parties deleted their accounts")
	if err != nil {
		t.Fatalf("resolving: %v", err)
	}
	if resolved.Status != "cancelled" {
		t.Errorf("status = %q, want cancelled", resolved.Status)
	}

	listing, err := f.db.GetListing(ctx, f.order.ListingID)
	if err != nil {
		t.Fatalf("reading the listing: %v", err)
	}
	if listing.Quantity != 10 {
		t.Errorf("quantity = %d, want 10 - the stock was not returned", listing.Quantity)
	}
}

func TestAStrandedOrderCannotBeResolvedAsCompleted(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()

	f.deleteAccount(t, f.buyer, "buyer")
	f.deleteAccount(t, f.seller, "seller")

	for _, outcome := range []string{"completed", "refunded"} {
		_, err := f.admins.Resolve(ctx, f.admin, f.order.ID, outcome, "a reason")

		var conflict *ConflictError
		if !errors.As(err, &conflict) {
			t.Errorf("resolving as %s: err = %#v, want *ConflictError - neither party ever acted", outcome, err)
		}
	}

	order, err := f.db.GetOrder(ctx, f.order.ID)
	if err != nil {
		t.Fatalf("re-reading: %v", err)
	}
	if order.Status != "pending" {
		t.Errorf("status = %q, want pending - a refused resolution still moved the order", order.Status)
	}
}

func TestOneDeletedPartyDoesNotStrandAnOrder(t *testing.T) {
	for _, gone := range []string{"buyer", "seller"} {
		t.Run("only the "+gone+" is deleted", func(t *testing.T) {
			f := newAdminOrderFixture(t)
			ctx := context.Background()

			leaving := f.buyer
			if gone == "seller" {
				leaving = f.seller
			}
			f.deleteAccount(t, leaving, gone)

			resolvable, err := f.db.GetOrderResolvability(ctx, f.order.ID)
			if err != nil {
				t.Fatalf("reading resolvability: %v", err)
			}
			if resolvable.Stranded || resolvable.Stuck {
				t.Errorf("stranded=%v stuck=%v, want both false - the other party can still cancel",
					resolvable.Stranded, resolvable.Stuck)
			}

			_, err = f.admins.Resolve(ctx, f.admin, f.order.ID, "cancelled", "a reason")

			var conflict *ConflictError
			if !errors.As(err, &conflict) {
				t.Errorf("err = %#v, want *ConflictError - a living party can still act", err)
			}
		})
	}
}

func TestAMarkedOrderIsNotStrandedWhenBothPartiesLeave(t *testing.T) {
	// Both marks, not just the seller's: the stranded arm tests each timestamp
	// separately, so one case is no evidence for the other.
	for _, mark := range []string{"seller_handover", "buyer_receipt"} {
		t.Run("after a "+mark, func(t *testing.T) {
			f := newAdminOrderFixture(t)
			ctx := context.Background()

			if _, err := f.svc.ConfirmOrder(ctx, f.seller, f.order.ID); err != nil {
				t.Fatalf("confirming: %v", err)
			}
			if mark == "seller_handover" {
				if _, err := f.svc.HandoverOrder(ctx, f.seller, f.order.ID); err != nil {
					t.Fatalf("handing over: %v", err)
				}
			} else {
				if _, err := f.svc.ReceiveOrder(ctx, f.buyer, f.order.ID); err != nil {
					t.Fatalf("confirming receipt: %v", err)
				}
			}

			f.deleteAccount(t, f.buyer, "buyer")
			f.deleteAccount(t, f.seller, "seller")

			resolvable, err := f.db.GetOrderResolvability(ctx, f.order.ID)
			if err != nil {
				t.Fatalf("reading resolvability: %v", err)
			}
			if resolvable.Stranded {
				t.Error("a marked order is flagged stranded - it belongs to the handshake shape, which waits seven days and allows every outcome")
			}
			if resolvable.Stuck {
				t.Error("stuck before the seven days are up")
			}
		})
	}
}

func TestATerminalOrderIsNeverStuck(t *testing.T) {
	// "cancelled" is the case that matters: cancelling is refused once a
	// handshake mark exists, so a cancelled order has neither timestamp set and
	// is told apart from a stranded one by its status alone.
	for _, want := range []string{"completed", "cancelled"} {
		t.Run("an order that is "+want, func(t *testing.T) {
			f := newAdminOrderFixture(t)
			ctx := context.Background()

			if want == "completed" {
				if _, err := f.svc.ConfirmOrder(ctx, f.seller, f.order.ID); err != nil {
					t.Fatalf("confirming: %v", err)
				}
				if _, err := f.svc.HandoverOrder(ctx, f.seller, f.order.ID); err != nil {
					t.Fatalf("handing over: %v", err)
				}
				if _, err := f.svc.ReceiveOrder(ctx, f.buyer, f.order.ID); err != nil {
					t.Fatalf("confirming receipt: %v", err)
				}
			} else {
				if _, err := f.svc.CancelOrder(ctx, f.buyer, f.order.ID); err != nil {
					t.Fatalf("cancelling: %v", err)
				}
			}

			f.deleteAccount(t, f.buyer, "buyer")
			f.deleteAccount(t, f.seller, "seller")

			order, err := f.db.GetOrder(ctx, f.order.ID)
			if err != nil {
				t.Fatalf("re-reading: %v", err)
			}
			if order.Status != want {
				t.Fatalf("the fixture produced %q, not %q", order.Status, want)
			}

			resolvable, err := f.db.GetOrderResolvability(ctx, f.order.ID)
			if err != nil {
				t.Fatalf("reading resolvability: %v", err)
			}
			if resolvable.Stranded || resolvable.Stuck {
				t.Errorf("stranded=%v stuck=%v, want both false - a finished order needs no rescue",
					resolvable.Stranded, resolvable.Stuck)
			}
		})
	}
}

func TestTheSevenDayWaitIsSevenDays(t *testing.T) {
	// An hour either side of the boundary rather than days: with only an
	// 8-day case, any threshold between 0 and 8 days passes.
	for _, c := range []struct {
		age       string
		wantStuck bool
	}{
		{"6 days 23 hours", false},
		{"7 days 1 hour", true},
	} {
		t.Run("a handover "+c.age+" old", func(t *testing.T) {
			f := newAdminOrderFixture(t)
			ctx := context.Background()

			if _, err := f.svc.ConfirmOrder(ctx, f.seller, f.order.ID); err != nil {
				t.Fatalf("confirming: %v", err)
			}
			if _, err := f.svc.HandoverOrder(ctx, f.seller, f.order.ID); err != nil {
				t.Fatalf("handing over: %v", err)
			}
			if _, err := f.db.ExecContext(ctx,
				`UPDATE orders SET seller_handed_over_at = now() - $2::interval WHERE id = $1`,
				f.order.ID, c.age); err != nil {
				t.Fatalf("backdating: %v", err)
			}

			resolvable, err := f.db.GetOrderResolvability(ctx, f.order.ID)
			if err != nil {
				t.Fatalf("reading resolvability: %v", err)
			}
			if resolvable.Stuck != c.wantStuck {
				t.Errorf("stuck = %v, want %v for a handover %s old", resolvable.Stuck, c.wantStuck, c.age)
			}
		})
	}
}

func TestAListedAdminOrderCarriesTheWholeOrder(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()

	page, err := f.admins.List(ctx, dtos.AdminOrderQuery{})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("listed %d orders, want 1", len(page.Items))
	}

	got, want := page.Items[0], f.order
	if got.ID != want.ID {
		t.Errorf("id = %v, want %v", got.ID, want.ID)
	}
	if got.BuyerID != want.BuyerID.String() {
		t.Errorf("buyer = %v, want %v", got.BuyerID, want.BuyerID)
	}
	if got.SellerID != want.SellerID.String() {
		t.Errorf("seller = %v, want %v - buyer and seller are the pair most easily swapped", got.SellerID, want.SellerID)
	}
	if got.ListingID != want.ListingID {
		t.Errorf("listing = %v, want %v", got.ListingID, want.ListingID)
	}
	if got.ListingTitle != want.ListingTitle {
		t.Errorf("listing title = %q, want %q", got.ListingTitle, want.ListingTitle)
	}
	if got.Quantity != want.Quantity {
		t.Errorf("quantity = %d, want %d", got.Quantity, want.Quantity)
	}
	if got.Status != want.Status {
		t.Errorf("status = %q, want %q", got.Status, want.Status)
	}
	if got.UnitPrice != 18.10 || got.TotalPrice != 36.20 {
		t.Errorf("prices = %v / %v, want 18.1 / 36.2", got.UnitPrice, got.TotalPrice)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at did not survive the view")
	}
}

// Do not delete: the view expands "o.*" at creation time, so a column added to
// orders later never reaches admin_orders, and sqlc freezes the same shape.
// Nothing else fails - the field is simply published as a zero value. This is
// what turns that silence into a red test.
func TestTheAdminOrdersViewCarriesEveryOrderColumn(t *testing.T) {
	order := reflect.TypeOf(database.Order{})
	admin := reflect.TypeOf(database.AdminOrder{})

	for i := range order.NumField() {
		field := order.Field(i)

		got, ok := admin.FieldByName(field.Name)
		if !ok {
			t.Errorf("database.Order.%s is missing from the admin_orders view - add it to migration 016 and recreate the view",
				field.Name)
			continue
		}
		if got.Type != field.Type {
			t.Errorf("%s is %v on the view and %v on the table", field.Name, got.Type, field.Type)
		}
	}
}

func TestASuspendedPairIsNotStranded(t *testing.T) {
	f := newAdminOrderFixture(t)
	ctx := context.Background()

	// Suspended directly rather than through the admin service: this is about
	// what the view reads, and the fixture should not depend on how a
	// suspension is applied.
	for _, id := range []uuid.UUID{f.buyer, f.seller} {
		if _, err := f.db.ExecContext(ctx,
			`UPDATE users SET suspended_at = now() WHERE id = $1`, id); err != nil {
			t.Fatalf("suspending: %v", err)
		}
	}

	resolvable, err := f.db.GetOrderResolvability(ctx, f.order.ID)
	if err != nil {
		t.Fatalf("reading resolvability: %v", err)
	}

	// An order between two suspended users is just as unreachable as one
	// between two deleted ones, and the predicate reads deleted_at anyway.
	// That is the decision, not an oversight: suspension is reversible, so the
	// order can become actionable again, while deletion cannot be undone.
	if resolvable.Stranded {
		t.Error("a suspended pair was flagged stranded - suspension is reversible, so the order is not permanently stuck")
	}
	if resolvable.Stuck {
		t.Error("a suspended pair made the order stuck")
	}

	_, err = f.admins.Resolve(ctx, f.admin, f.order.ID, "cancelled", "both suspended")

	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("resolve error = %v, want a conflict - an admin must not end an order that can still move", err)
	}
}
