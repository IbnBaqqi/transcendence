// Package app wires up dependencies and defines HTTP routes for the transcendence server.
package app

import (
	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/config"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/storage"
)

// api holds all dependencies for the api handlers
type api struct {
	DB           *database.DB
	JWT          *auth.JwtService
	Auth         *auth.Service
	Listing      *service.ListingService
	Order        *service.OrderService
	Saved        *service.SavedListingService
	Conversation *service.ConversationService
	User         *service.UserService
	ListingImage *service.ListingImageService
	Files        *storage.Local
	Upload       config.UploadConfig
}

// New initializes all services and returns a pointer to api
func New(cfg *config.Config, db *database.DB) (*api, error) {
	files, err := storage.NewLocal(cfg.Upload.Dir)
	if err != nil {
		return nil, err
	}

	jwtService := auth.NewJwtService(cfg.Auth.JWTSecret)
	authService := auth.NewService(db.Queries, jwtService)
	listingService := service.NewListingService(db, files) // needs *DB for transaction
	orderService := service.NewOrderService(db)
	savedService := service.NewSavedListingService(db.Queries)
	conversationService := service.NewConversationService(db)
	userService := service.NewUserService(db.Queries)
	listingImageService := service.NewListingImageService(db, files, cfg.Upload.MaxPerListing)

	return &api{
		DB:           db,
		JWT:          jwtService,
		Auth:         authService,
		Listing:      listingService,
		Order:        orderService,
		Saved:        savedService,
		Conversation: conversationService,
		User:         userService,
		ListingImage: listingImageService,
		Files:        files,
		Upload:       cfg.Upload,
	}, nil
}
