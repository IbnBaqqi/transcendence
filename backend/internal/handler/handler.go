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
	Follow         *service.FollowService
	ListingImage   *service.ListingImageService
	maxUploadBytes int64
}

func New(
	db *database.DB,
	authService *auth.Service,
	listingService *service.ListingService,
	orderService *service.OrderService,
	savedService *service.SavedListingService,
	conversationService *service.ConversationService,
	userService *service.UserService,
	followService *service.FollowService,
	listingImageService *service.ListingImageService,
	maxUploadBytes int64,
) *Handler {
	return &Handler{
		db:             db,
		Auth:           authService,
		Listing:        listingService,
		Order:          orderService,
		Saved:          savedService,
		Conversation:   conversationService,
		User:           userService,
		Follow:         followService,
		ListingImage:   listingImageService,
		maxUploadBytes: maxUploadBytes,
	}
}
