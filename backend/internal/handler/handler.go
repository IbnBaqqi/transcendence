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
	Saved   *service.SavedListingService
}

func New(
	db *database.DB,
	authService *auth.Service,
	listingService *service.ListingService,
	savedService *service.SavedListingService,
) *Handler {
	return &Handler{
		db:      db,
		Auth:    authService,
		Listing: listingService,
		Saved:   savedService,
	}
}
