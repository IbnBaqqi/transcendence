package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

type reviewFixture struct {
	reviews  *ReviewService
	orders   *OrderService
	users    *UserService
	profiles *ProfileService
	db       *database.DB
	seller   uuid.UUID
	buyer    uuid.UUID
	other    uuid.UUID
	listing  uuid.UUID
}

func newReviewFixture(t *testing.T) reviewFixture {
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
	seller, buyer, other := mk("seller"), mk("buyer"), mk("other")

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

	return reviewFixture{
		reviews:  NewReviewService(db.Queries),
		orders:   NewOrderService(db, notify.Disabled{}),
		users:    NewUserService(db, files),
		profiles: NewProfileService(db, files),
		db:       db,
		seller:   seller,
		buyer:    buyer,
		other:    other,
		listing:  listing.ID,
	}
}

// completedOrder drives a real order to 'completed' rather than writing the
// status directly - the point of anchoring on an order is that it took both
// parties, so the fixture has to do that too.
func (f reviewFixture) completedOrder(t *testing.T) database.Order {
	t.Helper()
	ctx := context.Background()

	order, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
		ListingID: f.listing, Quantity: 1,
	})
	if err != nil {
		t.Fatalf("ordering: %v", err)
	}
	if _, err := f.orders.ConfirmOrder(ctx, f.seller, order.ID); err != nil {
		t.Fatalf("confirming: %v", err)
	}
	if _, err := f.orders.HandoverOrder(ctx, f.seller, order.ID); err != nil {
		t.Fatalf("handing over: %v", err)
	}
	done, err := f.orders.ReceiveOrder(ctx, f.buyer, order.ID)
	if err != nil {
		t.Fatalf("receiving: %v", err)
	}
	if done.Status != "completed" {
		t.Fatalf("order status = %q, want completed", done.Status)
	}
	return done
}

func TestAReviewNeedsACompletedOrder(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	order, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
		ListingID: f.listing, Quantity: 1,
	})
	if err != nil {
		t.Fatalf("ordering: %v", err)
	}

	var conflict *ConflictError
	if _, err := f.reviews.Create(ctx, f.buyer, order.ID, 5, ""); !errors.As(err, &conflict) {
		t.Errorf("reviewing a pending order: err = %#v, want *ConflictError", err)
	}
}

func TestOnlyTheBuyerCanReview(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	order := f.completedOrder(t)

	// 404, not 403: someone with no part in an order should not learn it exists.
	for _, who := range []uuid.UUID{f.seller, f.other} {
		if _, err := f.reviews.Create(ctx, who, order.ID, 5, ""); !isNotFound(err) {
			t.Errorf("reviewing someone else's order: err = %#v, want *NotFoundError", err)
		}
	}
}

func TestOneReviewPerOrder(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	order := f.completedOrder(t)

	if _, err := f.reviews.Create(ctx, f.buyer, order.ID, 5, "good chanterelles"); err != nil {
		t.Fatalf("first review: %v", err)
	}

	var conflict *ConflictError
	if _, err := f.reviews.Create(ctx, f.buyer, order.ID, 1, "changed my mind"); !errors.As(err, &conflict) {
		t.Errorf("second review: err = %#v, want *ConflictError", err)
	}
}

// The service check is a read; two submissions race past it. The unique index
// is what actually holds, so it is what this test exercises.
func TestConcurrentReviewsCannotBothLand(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	order := f.completedOrder(t)

	var wg sync.WaitGroup
	errs := make([]error, 8)

	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = f.reviews.Create(ctx, f.buyer, order.ID, 5, "")
		}()
	}
	wg.Wait()

	created := 0
	for _, err := range errs {
		if err == nil {
			created++
		}
	}
	if created != 1 {
		t.Errorf("%d of 8 concurrent reviews landed, want exactly 1", created)
	}

	rating, err := f.reviews.RatingFor(ctx, f.seller)
	if err != nil {
		t.Fatalf("reading the rating: %v", err)
	}
	if rating.Total != 1 {
		t.Errorf("review count = %d, want 1", rating.Total)
	}
}

func TestAnEditClearsTheSameBarAsTheOriginal(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	order := f.completedOrder(t)
	review, err := f.reviews.Create(ctx, f.buyer, order.ID, 5, "good")
	if err != nil {
		t.Fatalf("creating: %v", err)
	}

	var invalid *ValidationError
	if _, err := f.reviews.Update(ctx, f.buyer, review.ID, 7, "good"); !errors.As(err, &invalid) {
		t.Errorf("editing to an out-of-range rating: err = %#v, want *ValidationError", err)
	}

	if _, err := f.reviews.Update(ctx, f.other, review.ID, 1, "not theirs"); !isNotFound(err) {
		t.Errorf("editing someone else's review: err = %#v, want *NotFoundError", err)
	}

	updated, err := f.reviews.Update(ctx, f.buyer, review.ID, 3, "on reflection")
	if err != nil {
		t.Fatalf("editing: %v", err)
	}
	if updated.Rating != 3 || updated.Comment.String != "on reflection" {
		t.Errorf("the edit did not apply: %+v", updated)
	}
	if !updated.UpdatedAt.After(review.UpdatedAt) {
		t.Error("updated_at did not move, so an edited review is indistinguishable")
	}
}

func TestTheAverageReflectsEveryReviewAndItsEdits(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	order := f.completedOrder(t)
	review, err := f.reviews.Create(ctx, f.buyer, order.ID, 5, "")
	if err != nil {
		t.Fatalf("creating: %v", err)
	}

	rating, err := f.reviews.RatingFor(ctx, f.seller)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if rating.Average != 5 || rating.Total != 1 {
		t.Errorf("average = %v over %d, want 5 over 1", rating.Average, rating.Total)
	}

	if _, err := f.reviews.Update(ctx, f.buyer, review.ID, 1, ""); err != nil {
		t.Fatalf("editing: %v", err)
	}

	rating, err = f.reviews.RatingFor(ctx, f.seller)
	if err != nil {
		t.Fatalf("re-reading: %v", err)
	}
	if rating.Average != 1 {
		t.Errorf("average = %v after an edit, want 1 - it is computed, not stored", rating.Average)
	}
}

func TestAnUnratedSellerReadsAsZeroOverZero(t *testing.T) {
	f := newReviewFixture(t)

	rating, err := f.reviews.RatingFor(context.Background(), f.seller)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if rating.Average != 0 || rating.Total != 0 {
		t.Errorf("average = %v over %d, want 0 over 0 - AVG is NULL without the COALESCE",
			rating.Average, rating.Total)
	}
}

// The whole reason reviewer_id is SET NULL and the list uses a LEFT JOIN.
func TestAReviewOutlivesItsAuthor(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	order := f.completedOrder(t)
	if _, err := f.reviews.Create(ctx, f.buyer, order.ID, 4, "picked fresh"); err != nil {
		t.Fatalf("creating: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.buyer, "buyer"); err != nil {
		t.Fatalf("deleting the buyer: %v", err)
	}

	rows, err := f.reviews.ListForSeller(ctx, f.seller)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("reviews = %d, want 1 - an inner join would have dropped it", len(rows))
	}
	if rows[0].ReviewerID.Valid {
		t.Error("the review still names its author")
	}
	if rows[0].Comment.String != "picked fresh" {
		t.Error("the review's content did not survive")
	}

	// Still counted: a seller's average must not move for reasons that have
	// nothing to do with the seller.
	rating, err := f.reviews.RatingFor(ctx, f.seller)
	if err != nil {
		t.Fatalf("reading the rating: %v", err)
	}
	if rating.Total != 1 || rating.Average != 4 {
		t.Errorf("average = %v over %d, want 4 over 1", rating.Average, rating.Total)
	}
}

func TestACommentIsSanitisedAndCapped(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	order := f.completedOrder(t)

	var invalid *ValidationError
	if _, err := f.reviews.Create(ctx, f.buyer, order.ID, 5, strings.Repeat("a", 1001)); !errors.As(err, &invalid) {
		t.Errorf("an over-long comment: err = %#v, want *ValidationError", err)
	}

	review, err := f.reviews.Create(ctx, f.buyer, order.ID, 5, "fresh\u202e and good")
	if err != nil {
		t.Fatalf("creating: %v", err)
	}
	if strings.ContainsRune(review.Comment.String, '\u202e') {
		t.Errorf("a bidi override survived into stored text: %q", review.Comment.String)
	}
}
