package dtos

import (
	"github.com/IbnBaqqi/transcendence/internal/database"
)

type AdminOrderQuery struct {
	Status      string
	CreatedFrom string
	CreatedTo   string
	Stuck       string
	Page        string
	Limit       string
}

type AdminOrderResponse struct {
	OrderResponse
	Stuck bool `json:"stuck"`
}

type PaginatedAdminOrders struct {
	Items      []AdminOrderResponse `json:"items"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	TotalPages int                  `json:"total_pages"`
}

type ResolveOrderRequest struct {
	Outcome string `json:"outcome"`
	Reason  string `json:"reason"`
}

func ToAdminOrderResponse(o database.Order, stuck bool) AdminOrderResponse {
	return AdminOrderResponse{
		OrderResponse: NewOrderResponse(o),
		Stuck:         stuck,
	}
}
