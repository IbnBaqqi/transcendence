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
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// Nothing exercised the HTTP layer for reviews, which is exactly why the write
// endpoints shipped returning an empty reviewer name.
func TestCreatingAReviewEchoesTheAuthor(t *testing.T) {
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

	orders := service.NewOrderService(db, notify.Disabled{})
	order, err := orders.CreateOrder(ctx, buyer, dtos.CreateOrderInput{
		ListingID: listing.ID, Quantity: 1,
	})
	if err != nil {
		t.Fatalf("ordering: %v", err)
	}
	for _, step := range []func() error{
		func() error { _, err := orders.ConfirmOrder(ctx, seller, order.ID); return err },
		func() error { _, err := orders.HandoverOrder(ctx, seller, order.ID); return err },
		func() error { _, err := orders.ReceiveOrder(ctx, buyer, order.ID); return err },
	} {
		if err := step(); err != nil {
			t.Fatalf("completing the order: %v", err)
		}
	}

	h := New(Deps{DB: db, Review: service.NewReviewService(db.Queries)})

	req := httptest.NewRequest(http.MethodPost, "/orders/"+order.ID.String()+"/review",
		strings.NewReader(`{"rating":5,"comment":"picked fresh"}`))
	req.Header.Set("Content-Type", "application/json")

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", order.ID.String())
	reqCtx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	reqCtx = auth.WithUser(reqCtx, auth.User{ID: buyer, Name: "buyer", Role: auth.RoleUser})

	rec := httptest.NewRecorder()
	h.CreateReview(rec, req.WithContext(reqCtx))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"reviewer":"buyer"`) {
		t.Errorf("the response does not name its author:\n%s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"reviewer":""`) {
		t.Errorf("the author came back empty:\n%s", rec.Body.String())
	}
}
