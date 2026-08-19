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
	Conversation   *service.ConversationService
	User           *service.UserService
	Profile        *service.ProfileService
	ListingImage   *service.ListingImageService
	maxUploadBytes int64
	cookieSecure   bool
}

func New(
	db *database.DB,
	authService *auth.Service,
	listingService *service.ListingService,
	orderService *service.OrderService,
	savedService *service.SavedListingService,
	conversationService *service.ConversationService,
	userService *service.UserService,
	profileService *service.ProfileService,
	listingImageService *service.ListingImageService,
	maxUploadBytes int64,
	cookieSecure bool,
) *Handler {
	return &Handler{
		db:             db,
		Auth:           authService,
		Listing:        listingService,
		Order:          orderService,
		Saved:          savedService,
		Conversation:   conversationService,
		User:           userService,
		Profile:        profileService,
		ListingImage:   listingImageService,
		maxUploadBytes: maxUploadBytes,
		cookieSecure:   cookieSecure,
	}
}
