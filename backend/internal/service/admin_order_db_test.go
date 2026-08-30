package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
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
		admins:       NewAdminOrderService(f.db),
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

	order, err := f.db.GetOrder(ctx, f.order.ID)
	if err != nil {
		t.Fatalf("re-reading: %v", err)
	}
	if !isStuck(order) {
		t.Fatalf("the fixture did not produce a stuck order: status=%s handover=%v received=%v",
			order.Status, order.SellerHandedOverAt.Valid, order.BuyerReceivedAt.Valid)
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
		t.Error("the row the SQL filter selected is not flagged stuck by the Go predicate")
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
