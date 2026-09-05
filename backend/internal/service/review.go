package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

const (
	maxReviewComment = 1000

	reviewedConstraint = "reviews_order_id_uq"
)

type ReviewService struct {
	db *database.DB
}

func NewReviewService(db *database.DB) *ReviewService {
	return &ReviewService{db: db}
}

func validateRating(rating int32) error {
	if rating == 0 {
		return &ValidationError{Message: "Rating is required"}
	}
	if rating < 1 || rating > 5 {
		return &ValidationError{Message: "Rating must be between 1 and 5"}
	}
	return nil
}

// Both are shared by create and update: an edit has to clear the same bar as
// the original, or the rules are only enforced on the way in.
func validateComment(comment string) (string, error) {
	if !utf8.ValidString(comment) || strings.ContainsRune(comment, 0) {
		return "", &ValidationError{Message: "Comment must be valid UTF-8 without null bytes"}
	}

	comment = sanitizeFreeText(comment)

	if utf8.RuneCountInString(comment) > maxReviewComment {
		return "", &ValidationError{Message: "Comment is too long"}
	}

	return comment, nil
}

func (s *ReviewService) Create(
	ctx context.Context,
	buyerID uuid.UUID,
	orderID uuid.UUID,
	rating int32,
	comment string,
) (database.Review, error) {
	if err := validateRating(rating); err != nil {
		return database.Review{}, err
	}

	comment, err := validateComment(comment)
	if err != nil {
		return database.Review{}, err
	}

	order, err := s.db.GetOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Review{}, &NotFoundError{Message: "Order not found"}
		}
		return database.Review{}, err
	}

	if order.BuyerID != buyerID {
		return database.Review{}, &NotFoundError{Message: "Order not found"}
	}

	// The subject has to still exist. GET /users/{id} 404s a departed seller
	// and CreateOrder refuses to sell from one, so adding new public commentary
	// about an account nobody can view would be the odd one out.
	seller, err := s.db.GetUser(ctx, order.SellerID)
	if err != nil {
		return database.Review{}, err
	}
	if seller.DeletedAt.Valid {
		return database.Review{}, &NotFoundError{Message: "Order not found"}
	}

	if order.Status != "completed" {
		return database.Review{}, &ConflictError{
			Message: "You can review an order once it is completed",
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return database.Review{}, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("review transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	review, err := qtx.CreateReview(ctx, database.CreateReviewParams{
		ID:         database.NewID(),
		OrderID:    orderID,
		SellerID:   order.SellerID,
		ReviewerID: uuid.NullUUID{UUID: buyerID, Valid: true},
		Rating:     rating,
		Comment:    sql.NullString{String: comment, Valid: comment != ""},
	})

	if isUniqueViolation(err, reviewedConstraint) {
		return database.Review{}, &ConflictError{Message: "You have already reviewed this order"}
	}
	if err != nil {
		return database.Review{}, err
	}

	// actor_id is the seller here, not the buyer who wrote the review: it is
	// what the row points at, and a seller's reviews live on their own profile.
	// Setting it to the reviewer sends them to a page their review is not on.
	if err := recordActorNotification(ctx, qtx, order.SellerID, notifyKindReviewReceived, order.SellerID); err != nil {
		return database.Review{}, err
	}

	if err := tx.Commit(); err != nil {
		return database.Review{}, err
	}

	return review, nil
}

func (s *ReviewService) Update(
	ctx context.Context,
	reviewerID uuid.UUID,
	reviewID uuid.UUID,
	rating int32,
	comment dtos.OptionalString,
) (database.Review, error) {
	if err := validateRating(rating); err != nil {
		return database.Review{}, err
	}

	// An absent comment leaves the stored one alone; an explicit null clears it.
	params := database.UpdateReviewParams{
		ID:         reviewID,
		ReviewerID: reviewerID,
		Rating:     rating,
		CommentSet: comment.Set,
	}

	if comment.Set && comment.Value != nil {
		cleaned, err := validateComment(*comment.Value)
		if err != nil {
			return database.Review{}, err
		}
		params.Comment = sql.NullString{String: cleaned, Valid: cleaned != ""}
	}

	review, err := s.db.UpdateReview(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Review{}, &NotFoundError{Message: "Review not found"}
		}
		return database.Review{}, err
	}

	return review, nil
}

func (s *ReviewService) GetForOrder(
	ctx context.Context,
	userID, orderID uuid.UUID,
) (database.Review, string, error) {
	order, err := s.db.GetOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Review{}, "", &NotFoundError{Message: "Review not found"}
		}
		return database.Review{}, "", err
	}

	if userID != order.BuyerID && userID != order.SellerID {
		return database.Review{}, "", &NotFoundError{Message: "Review not found"}
	}

	review, err := s.db.GetReviewForOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Review{}, "", &NotFoundError{Message: "Review not found"}
		}
		return database.Review{}, "", err
	}

	name := dtos.DeletedUserName
	if review.ReviewerID.Valid {
		reviewer, err := s.db.GetUser(ctx, review.ReviewerID.UUID)
		if err == nil && !reviewer.DeletedAt.Valid {
			name = reviewer.Username
		}
	}

	return review, name, nil
}

func (s *ReviewService) ListForSeller(
	ctx context.Context,
	sellerID uuid.UUID,
	q dtos.ReviewQuery,
) (dtos.PaginatedReviews, error) {
	seller, err := s.db.GetUser(ctx, sellerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dtos.PaginatedReviews{}, &NotFoundError{Message: "User not found"}
		}
		return dtos.PaginatedReviews{}, err
	}
	if seller.DeletedAt.Valid {
		return dtos.PaginatedReviews{}, &NotFoundError{Message: "User not found"}
	}

	paging, err := parsePaging(q.Page, q.Limit)
	if err != nil {
		return dtos.PaginatedReviews{}, err
	}

	total, err := s.db.CountReviewsForSeller(ctx, sellerID)
	if err != nil {
		return dtos.PaginatedReviews{}, err
	}

	rows, err := s.db.ListReviewsForSeller(ctx, database.ListReviewsForSellerParams{
		SellerID: sellerID, PageLimit: paging.pageLimit, PageOffset: paging.pageOffset,
	})
	if err != nil {
		return dtos.PaginatedReviews{}, err
	}

	return dtos.PaginatedReviews{
		Items:      dtos.ToReviewResponses(rows),
		Total:      total,
		Page:       paging.page,
		Limit:      paging.limit,
		TotalPages: int((total + int64(paging.limit) - 1) / int64(paging.limit)),
	}, nil
}

func (s *ReviewService) RatingFor(ctx context.Context, sellerID uuid.UUID) (database.SellerRatingRow, error) {
	return s.db.SellerRating(ctx, sellerID)
}
