package service

import (
	"context"
	"database/sql"
	"errors"
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
	db *database.Queries
}

func NewReviewService(db *database.Queries) *ReviewService {
	return &ReviewService{db: db}
}

func validateRating(rating int32) error {
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

	comment = sanitizeReportDetail(comment)

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

	review, err := s.db.CreateReview(ctx, database.CreateReviewParams{
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

// ListForSeller answers 404 for a departed seller, matching GET /users/{id}:
// reviews describe a person, and serving commentary about a profile nobody can
// open would leave it visible with no way to see who it is about.
// GetForOrder reads back the review on an order, which is also what a buyer
// needs to see their own.
func (s *ReviewService) GetForOrder(ctx context.Context, orderID uuid.UUID) (database.Review, error) {
	review, err := s.db.GetReviewForOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Review{}, &NotFoundError{Message: "Review not found"}
		}
		return database.Review{}, err
	}
	return review, nil
}

func (s *ReviewService) ListForSeller(ctx context.Context, sellerID uuid.UUID) ([]database.ListReviewsForSellerRow, error) {
	seller, err := s.db.GetUser(ctx, sellerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &NotFoundError{Message: "User not found"}
		}
		return nil, err
	}
	if seller.DeletedAt.Valid {
		return nil, &NotFoundError{Message: "User not found"}
	}

	return s.db.ListReviewsForSeller(ctx, sellerID)
}

func (s *ReviewService) RatingFor(ctx context.Context, sellerID uuid.UUID) (database.SellerRatingRow, error) {
	return s.db.SellerRating(ctx, sellerID)
}
