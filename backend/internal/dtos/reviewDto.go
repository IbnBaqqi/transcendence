package dtos

import (
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

// Comment is optional on a PATCH, so absent has to mean "leave it alone".
// A plain string cannot say that - an omitted key and "" look identical, and
// the update would wipe text the author never asked to remove. Reviews are
// not deletable and keep no history, so that loss is permanent.
type ReviewRequest struct {
	Rating  int32          `json:"rating"`
	Comment OptionalString `json:"comment"`
}

type ReviewResponse struct {
	ID        uuid.UUID `json:"id"`
	OrderID   uuid.UUID `json:"order_id"`
	Rating    int32     `json:"rating"`
	Comment   string    `json:"comment"`
	Reviewer  string    `json:"reviewer"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToReviewResponse(r database.Review, reviewer string) ReviewResponse {
	return ReviewResponse{
		ID:        r.ID,
		OrderID:   r.OrderID,
		Rating:    r.Rating,
		Comment:   r.Comment.String,
		Reviewer:  reviewer,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func ToReviewResponses(rows []database.ListReviewsForSellerRow) []ReviewResponse {
	out := make([]ReviewResponse, 0, len(rows))
	for _, r := range rows {
		name := deletedUserName
		if r.ReviewerID.Valid && !r.ReviewerDeletedAt.Valid {
			name = r.ReviewerUsername.String
		}

		out = append(out, ReviewResponse{
			ID:        r.ID,
			OrderID:   r.OrderID,
			Rating:    r.Rating,
			Comment:   r.Comment.String,
			Reviewer:  name,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		})
	}
	return out
}
