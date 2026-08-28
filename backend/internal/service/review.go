package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
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

func validateReview(rating int32, comment string) (string, error) {
	if rating < 1 || rating > 5 {
		return "", &ValidationError{Message: "Rating must be between 1 and 5"}
	}

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
	comment, err := validateReview(rating, comment)
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
	comment string,
) (database.Review, error) {
	comment, err := validateReview(rating, comment)
	if err != nil {
		return database.Review{}, err
	}

	review, err := s.db.UpdateReview(ctx, database.UpdateReviewParams{
		ID:         reviewID,
		ReviewerID: reviewerID,
		Rating:     rating,
		Comment:    sql.NullString{String: comment, Valid: comment != ""},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Review{}, &NotFoundError{Message: "Review not found"}
		}
		return database.Review{}, err
	}

	return review, nil
}

func (s *ReviewService) ListForSeller(ctx context.Context, sellerID uuid.UUID) ([]database.ListReviewsForSellerRow, error) {
	return s.db.ListReviewsForSeller(ctx, sellerID)
}

func (s *ReviewService) RatingFor(ctx context.Context, sellerID uuid.UUID) (database.SellerRatingRow, error) {
	return s.db.SellerRating(ctx, sellerID)
}
