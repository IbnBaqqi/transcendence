package handler

import (
	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/oauth"
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
	Follow         *service.FollowService
	Block          *service.BlockService
	APIKey         *service.APIKeyService
	ListingImage   *service.ListingImageService
	maxUploadBytes int64
	cookieSecure   bool
	oauth          *oauth.Registry
	frontendURL    string
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
	followService *service.FollowService,
	blockService *service.BlockService,
	apiKeyService *service.APIKeyService,
	listingImageService *service.ListingImageService,
	maxUploadBytes int64,
	cookieSecure bool,
	oauthRegistry *oauth.Registry,
	frontendURL string,
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
		Follow:         followService,
		Block:          blockService,
		APIKey:         apiKeyService,
		ListingImage:   listingImageService,
		maxUploadBytes: maxUploadBytes,
		cookieSecure:   cookieSecure,
		oauth:          oauthRegistry,
		frontendURL:    frontendURL,
	}
}
