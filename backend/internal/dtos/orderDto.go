package dtos

import (
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

// CreateOrderInput is the JSON body for POST /orders.
type CreateOrderInput struct {
	ListingID uuid.UUID `json:"listing_id"`
	Quantity  int32     `json:"quantity"`
}

// OrderResponse is the shape we send back to clients.
type OrderResponse struct {
	ID                 uuid.UUID  `json:"id"`
	ListingID          uuid.UUID  `json:"listing_id"`
	ListingTitle       string     `json:"listing_title"`
	BuyerID            string     `json:"buyer_id"`
	SellerID           string     `json:"seller_id"`
	Quantity           int32      `json:"quantity"`
	UnitPrice          float64    `json:"unit_price"`
	TotalPrice         float64    `json:"total_price"`
	Status             string     `json:"status"`
	SellerHandedOverAt *time.Time `json:"seller_handed_over_at"`
	BuyerReceivedAt    *time.Time `json:"buyer_received_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// NewOrderResponse converts one database row into the API shape.
func NewOrderResponse(o database.Order) OrderResponse {
	return OrderResponse{
		ID:                 o.ID,
		ListingID:          o.ListingID,
		ListingTitle:       o.ListingTitle,
		BuyerID:            o.BuyerID.String(),
		SellerID:           o.SellerID.String(),
		Quantity:           o.Quantity,
		UnitPrice:          numericToFloat(o.UnitPrice),
		TotalPrice:         numericToFloat(o.TotalPrice),
		Status:             o.Status,
		SellerHandedOverAt: nullTimeToPtr(o.SellerHandedOverAt),
		BuyerReceivedAt:    nullTimeToPtr(o.BuyerReceivedAt),
		CreatedAt:          o.CreatedAt.Time,
		UpdatedAt:          o.UpdatedAt.Time,
	}
}

// nullTimeToPtr converts sqlc's sql.NullTime into a *time.Time.
func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}

// NewOrderResponse converts a whole list.
func NewOrderResponses(orders []database.Order) []OrderResponse {
	out := make([]OrderResponse, 0, len(orders))
	for _, o := range orders {
		out = append(out, NewOrderResponse(o))
	}
	return out
}
