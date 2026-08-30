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

func (f reviewFixture) completedOrderFor(t *testing.T, buyer uuid.UUID) database.Order {
	t.Helper()
	ctx := context.Background()

	order, err := f.orders.CreateOrder(ctx, buyer, dtos.CreateOrderInput{
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
	done, err := f.orders.ReceiveOrder(ctx, buyer, order.ID)
	if err != nil {
		t.Fatalf("receiving: %v", err)
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

// "Completed" means both parties acted. Handed over but not yet received is
// the state that distinguishes that from a seller declaring it done alone.
func TestAHandoverAloneDoesNotAllowAReview(t *testing.T) {
	f := newReviewFixture(t)
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
	handed, err := f.orders.HandoverOrder(ctx, f.seller, order.ID)
	if err != nil {
		t.Fatalf("handing over: %v", err)
	}
	if handed.Status == "completed" {
		t.Fatal("a handover alone completed the order, so this test proves nothing")
	}

	var conflict *ConflictError
	if _, err := f.reviews.Create(ctx, f.buyer, order.ID, 5, ""); !errors.As(err, &conflict) {
		t.Errorf("reviewing a handed-over order: err = %#v, want *ConflictError", err)
	}
}

func TestACancelledOrderCannotBeReviewed(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	order, err := f.orders.CreateOrder(ctx, f.buyer, dtos.CreateOrderInput{
		ListingID: f.listing, Quantity: 1,
	})
	if err != nil {
		t.Fatalf("ordering: %v", err)
	}
	if _, err := f.orders.CancelOrder(ctx, f.buyer, order.ID); err != nil {
		t.Fatalf("cancelling: %v", err)
	}

	var conflict *ConflictError
	if _, err := f.reviews.Create(ctx, f.buyer, order.ID, 5, ""); !errors.As(err, &conflict) {
		t.Errorf("reviewing a cancelled order: err = %#v, want *ConflictError", err)
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

// There is no service-level "already reviewed?" check - the insert is the only
// gate, and the unique index is what makes it hold. This test is what says so.
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
	if _, err := f.reviews.Update(ctx, f.buyer, review.ID, 7, dtos.SetString("good")); !errors.As(err, &invalid) {
		t.Errorf("editing to an out-of-range rating: err = %#v, want *ValidationError", err)
	}

	if _, err := f.reviews.Update(ctx, f.other, review.ID, 1, dtos.SetString("not theirs")); !isNotFound(err) {
		t.Errorf("editing someone else's review: err = %#v, want *NotFoundError", err)
	}

	updated, err := f.reviews.Update(ctx, f.buyer, review.ID, 3, dtos.SetString("on reflection"))
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

	if _, err := f.reviews.Update(ctx, f.buyer, review.ID, 1, dtos.OptionalString{}); err != nil {
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

// Every other test has exactly one review, where AVG equals the single value
// and a broken aggregate is invisible. This one has two, and checks a second
// seller's reviews do not bleed into the first's.
func TestTheAverageIsPerSellerAndOverEveryReview(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	first := f.completedOrder(t)
	if _, err := f.reviews.Create(ctx, f.buyer, first.ID, 4, "solid"); err != nil {
		t.Fatalf("first review: %v", err)
	}

	second := f.completedOrderFor(t, f.other)
	if _, err := f.reviews.Create(ctx, f.other, second.ID, 5, "excellent"); err != nil {
		t.Fatalf("second review: %v", err)
	}

	rating, err := f.reviews.RatingFor(ctx, f.seller)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if rating.Total != 2 {
		t.Fatalf("count = %d, want 2", rating.Total)
	}
	if rating.Average != 4.5 {
		t.Errorf("average = %v, want 4.5", rating.Average)
	}

	// A seller with no reviews of their own must not inherit anyone else's.
	other, err := f.reviews.RatingFor(ctx, f.buyer)
	if err != nil {
		t.Fatalf("reading the buyer's rating: %v", err)
	}
	if other.Total != 0 {
		t.Errorf("an unrelated user has %d reviews - the seller_id filter is not holding", other.Total)
	}
}

// The "Deleted user" string is what the whole outlives-its-author design is
// for, and until now nothing asserted it past the database row.
func TestADepartedAuthorRendersAsDeletedUser(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	order := f.completedOrder(t)
	if _, err := f.reviews.Create(ctx, f.buyer, order.ID, 4, "picked fresh"); err != nil {
		t.Fatalf("creating: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.buyer, "buyer"); err != nil {
		t.Fatalf("deleting: %v", err)
	}

	rows, err := f.reviews.ListForSeller(ctx, f.seller)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}

	out := dtos.ToReviewResponses(rows)
	if len(out) != 1 {
		t.Fatalf("responses = %d, want 1", len(out))
	}
	if out[0].Reviewer != "Deleted user" {
		t.Errorf("reviewer = %q, want %q", out[0].Reviewer, "Deleted user")
	}
	if strings.Contains(out[0].Reviewer, "deleted-") {
		t.Errorf("the machine placeholder leaked to a client: %q", out[0].Reviewer)
	}
	if out[0].Comment != "picked fresh" {
		t.Errorf("comment = %q, want it intact", out[0].Comment)
	}
}

// The reason Comment is an OptionalString: a rating fix must not destroy text
// the author never touched, and reviews keep no history to recover it from.
func TestEditingTheRatingAloneKeepsTheComment(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	order := f.completedOrder(t)
	if _, err := f.reviews.Create(ctx, f.buyer, order.ID, 1, "a paragraph they wrote"); err != nil {
		t.Fatalf("creating: %v", err)
	}

	review, err := f.reviews.GetForOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}

	// A body of {"rating": 4} - comment absent, not empty.
	updated, err := f.reviews.Update(ctx, f.buyer, review.ID, 4, dtos.OptionalString{})
	if err != nil {
		t.Fatalf("editing the rating: %v", err)
	}

	if updated.Rating != 4 {
		t.Errorf("rating = %d, want 4", updated.Rating)
	}
	if updated.Comment.String != "a paragraph they wrote" {
		t.Errorf("comment = %q, want it untouched - an omitted field destroyed it", updated.Comment.String)
	}

	// An explicit null still clears it.
	cleared, err := f.reviews.Update(ctx, f.buyer, review.ID, 4, dtos.ClearString())
	if err != nil {
		t.Fatalf("clearing: %v", err)
	}
	if cleared.Comment.Valid {
		t.Errorf("an explicit null did not clear the comment: %q", cleared.Comment.String)
	}
}

func TestADepartedSellerHasNoPublicReviewPage(t *testing.T) {
	f := newReviewFixture(t)
	ctx := context.Background()

	order := f.completedOrder(t)
	if _, err := f.reviews.Create(ctx, f.buyer, order.ID, 5, "great mushrooms"); err != nil {
		t.Fatalf("creating: %v", err)
	}

	if err := f.users.DeleteAccount(ctx, f.seller, "seller"); err != nil {
		t.Fatalf("deleting the seller: %v", err)
	}

	// GET /users/{id} 404s for them, so their review page must too - otherwise
	// public commentary outlives the profile it describes.
	if _, err := f.reviews.ListForSeller(ctx, f.seller); !isNotFound(err) {
		t.Errorf("a departed seller still has a review page: err = %#v, want *NotFoundError", err)
	}
}
