package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func adminOrderHandler(t *testing.T) (*Handler, *database.DB, uuid.UUID) {
	t.Helper()

	db := testdb.New(t)

	admin, err := db.CreateUser(context.Background(), database.CreateUserParams{
		ID:       database.NewID(),
		Username: "admin", Email: "admin@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating the admin: %v", err)
	}

	return New(Deps{
		DB:         db,
		AdminOrder: service.NewAdminOrderService(db, notify.Disabled{}),
	}), db, admin.ID
}

func resolveAs(t *testing.T, h *Handler, caller uuid.UUID, idParam, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", idParam)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	ctx = auth.WithUser(ctx, auth.User{ID: caller, Role: auth.RoleAdmin})

	rec := httptest.NewRecorder()
	h.ResolveOrder(rec, req.WithContext(ctx))
	return rec
}

func TestResolvingAnOrderThatIsNotStuckIsAConflictNotAServerError(t *testing.T) {
	h, db, admin := adminOrderHandler(t)
	ctx := context.Background()

	seller, err := db.CreateUser(ctx, database.CreateUserParams{
		ID: database.NewID(), Username: "seller", Email: "seller@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating the seller: %v", err)
	}

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID: database.NewID(), SellerID: seller.ID, Title: "Chanterelles",
		Category: "mushrooms", Price: "18.10", Quantity: 5, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating a listing: %v", err)
	}

	order, err := db.CreateOrder(ctx, database.CreateOrderParams{
		ID: database.NewID(), ListingID: listing.ID, BuyerID: admin, SellerID: seller.ID,
		Quantity: 1, UnitPrice: "18.10", ListingTitle: "Chanterelles",
	})
	if err != nil {
		t.Fatalf("creating an order: %v", err)
	}

	rec := resolveAs(t, h, admin, order.ID.String(), `{"outcome":"refunded","reason":"why"}`)

	if rec.Code != http.StatusConflict {
		t.Fatalf("a pending order: got %d, want 409 - an unmapped service error is a 500", rec.Code)
	}
}

func TestResolvingRejectsAMalformedRequest(t *testing.T) {
	h, _, admin := adminOrderHandler(t)

	tests := []struct {
		name string
		id   string
		body string
	}{
		{"an unparseable id", "not-a-uuid", `{"outcome":"refunded","reason":"why"}`},
		{"an unknown field", uuid.New().String(), `{"outcome":"refunded","reason":"why","notify":true}`},
		{"not json at all", uuid.New().String(), `nonsense`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := resolveAs(t, h, admin, tt.id, tt.body); rec.Code != http.StatusBadRequest {
				t.Errorf("got %d, want 400", rec.Code)
			}
		})
	}
}

func TestResolvingWithoutAnAuthenticatedCallerIs401(t *testing.T) {
	h, _, _ := adminOrderHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"outcome":"refunded","reason":"why"}`))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", uuid.New().String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	h.ResolveOrder(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rec.Code)
	}
}

func TestTheAdminOrderListRejectsABadFilterWithA400(t *testing.T) {
	h, _, _ := adminOrderHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/?status=vanished", nil)
	rec := httptest.NewRecorder()
	h.ListOrdersForAdmin(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rec.Code)
	}
	if msg := decodeError(t, rec); !strings.Contains(msg, "Status must be") {
		t.Errorf("message = %q", msg)
	}
}
