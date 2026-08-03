package handler

import (
	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/service"
)

type Handler struct {
	db      *database.DB
	Auth    *auth.Service
	Listing *service.ListingService
	Order   *service.OrderService
}

func New(
	db *database.DB,
	authService *auth.Service,
	listingService *service.ListingService,
	orderService *service.OrderService,
) *Handler {
	return &Handler{
		db:      db,
		Auth:    authService,
		Listing: listingService,
		Order:   orderService,
	}
}
