package dtos

import (
	"database/sql"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

type CreateOrderInput struct {
	ListingID int32 `json:"listing_id"`
	Quantity  int32 `json:"quantity"`
}

type OrderResponse struct {
	ID                 int32      `json:"id"`
	ListingID          int32      `json:"listing_id"`
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

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}

func NewOrderResponses(orders []database.Order) []OrderResponse {
	out := make([]OrderResponse, 0, len(orders))
	for _, o := range orders {
		out = append(out, NewOrderResponse(o))
	}
	return out
}
