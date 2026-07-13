package handler

import (
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/service"
)

type Handler struct {
	db      *database.DB
	Listing *service.ListingService
}

func New(
	db *database.DB,
	listingService *service.ListingService,
) *Handler {
	return &Handler{
		db:      db,
		Listing: listingService,
	}
}
