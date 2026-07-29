package dtos

import (
	"time"
	"github.com/IbnBaqqi/transcendence/internal/database"
)
// --- Request DTOs ---

// CreateOrderInput is the JSON body for POST /orders.
//
// Note what is NOT here: seller_id, unit_price, total_price and status.
// A buyer must never be able to set those from the request - the service
// looks the listing up in the DB and delivers them. Only trust the client
// for "which listing" and "how many".
type CreateOrderInput struct {
	// `json:"..."` tags tell encoding/json which field maps to which
	// key in the request body (Go field names are capitalised, JSON keys aren't).
	ListingID int32 `json:"listing_id"`
	Quantity int32 `json:"quantity"`
}

// --- Response DTOs ---

// OrderResponse is the shape we send back to clients.
//
// Prices stay strings on purpose. The DB columns are NUMERIC(10, 2), with sqlc
// maps to string precisely because money in a float64 rounds badly. Parsing
// them into floats here would throw away the guarantee NUMERIC exists to give.
type OrderResponse struct {
	ID			int32		`json:"id"`
	ListingID	int32		`json:"listing_id"`
	BuyerID		string		`json:"buyer_id"`
	SellerID	string		`json:"seller_id"`
	Quantity	int32		`json:"quantity"`
	UnitPrice	string		`json:"unit_price"`
	TotalPrice	string		`json:"total_price"`
	Status		string		`json:"status"`
	CreatedAt	time.Time	`json:"created_at"`
	UpdatedAt	time.Time	`json:"updated_at"`
}

// NewOrderResponse converts one database row into the API shape.
// This is the only place that knows how the two differ, so handlers
// never have to think about sql.NullTime.
func NewOrderResponse(o database.Order) OrderResponse {
	return OrderResponse{
		ID:			o.ID,
		ListingID:	o.ListingID,
		BuyerID:	o.BuyerID.String(),
		SellerID:	o.SellerID.String(),
		Quantity:	o.Quantity,
		UnitPrice:	o.UnitPrice,
		TotalPrice:	o.TotalPrice,
		Status:		o.Status,
		CreatedAt:	o.CreatedAt.Time,
		UpdatedAt:	o.UpdatedAt.Time,
	}
}

// NewOrderResponse converts a whole list.
//
// The `make(...)` matters: a nil slice marshals to JSON `null`, but an empty
// slice marshals to `[]`. sqlc return s nil when a query matches now rows, so
// withouth this, "no orders yet" would send null and break clients doing
// orders.map(...).
func NewOrderResponses(orders []database.Order) []OrderResponse {
	out := make([]OrderResponse, 0, len(orders))
	for _, o := range orders {
		out = append(out, NewOrderResponse(o))
	}
	return out
}