package handler

import (
	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/service"
)

type Handler struct {
	db             *database.DB
	Auth           *auth.Service
	Listing        *service.ListingService
	Order          *service.OrderService
	Saved          *service.SavedListingService
	ListingImage   *service.ListingImageService
	maxUploadBytes int64
  User         *service.UserService
}

func New(
	db *database.DB,
	authService *auth.Service,
	listingService *service.ListingService,
	orderService *service.OrderService,
	savedService *service.SavedListingService,
  listingImageService *service.ListingImageService,
	maxUploadBytes int64,
	conversationService *service.ConversationService,
	userService *service.UserService,
) *Handler {
	return &Handler{
		db:           db,
		Auth:         authService,
		Listing:      listingService,
		Order:        orderService,
		Saved:        savedService,
		Conversation: conversationService,
		User:         userService,
    ListingImage:   listingImageService,
		maxUploadBytes: maxUploadBytes,
	}
}
