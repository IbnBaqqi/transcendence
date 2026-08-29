// Package app wires up dependencies and defines HTTP routes for the transcendence server.
package app

import (
	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/config"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/storage"
)

type api struct {
	DB           *database.DB
	JWT          *auth.JwtService
	Auth         *auth.Service
	Listing      *service.ListingService
	Order        *service.OrderService
	Saved        *service.SavedListingService
	Conversation *service.ConversationService
	User         *service.UserService
	Profile      *service.ProfileService
	Follow       *service.FollowService
	Block        *service.BlockService
	APIKey       *service.APIKeyService
	ListingImage *service.ListingImageService
	Report       *service.ReportService
	Moderation   *service.ModerationService
	AdminUser    *service.AdminUserService
	Files        *storage.Local
	Upload       config.UploadConfig
	AuthConfig   config.AuthConfig
}

func New(cfg *config.Config, db *database.DB, notifier notify.Notifier) (*api, error) {
	files, err := storage.NewLocal(cfg.Upload.Dir)
	if err != nil {
		return nil, err
	}

	jwtService := auth.NewJwtService(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL)
	authService := auth.NewService(db, jwtService, notifier, cfg.Auth.FrontendURL) // needs *DB for transaction
	listingService := service.NewListingService(db, files)                         // needs *DB for transaction
	orderService := service.NewOrderService(db, notifier)
	savedService := service.NewSavedListingService(db.Queries)
	conversationService := service.NewConversationService(db, notifier)
	userService := service.NewUserService(db, files)
	profileService := service.NewProfileService(db, files) // needs *DB for transaction
	followService := service.NewFollowService(db.Queries)
	blockService := service.NewBlockService(db.Queries)
	apiKeyService := service.NewAPIKeyService(db.Queries)
	listingImageService := service.NewListingImageService(db, files, cfg.Upload.MaxPerListing)
	reportService := service.NewReportService(db.Queries)
	adminUserService := service.NewAdminUserService(db, files)
	moderationService := service.NewModerationService(db, files) // needs *DB for transaction

	return &api{
		DB:           db,
		JWT:          jwtService,
		Auth:         authService,
		Listing:      listingService,
		Order:        orderService,
		Saved:        savedService,
		Conversation: conversationService,
		User:         userService,
		Profile:      profileService,
		Follow:       followService,
		Block:        blockService,
		APIKey:       apiKeyService,
		ListingImage: listingImageService,
		Report:       reportService,
		Moderation:   moderationService,
		AdminUser:    adminUserService,
		Files:        files,
		Upload:       cfg.Upload,
		AuthConfig:   cfg.Auth,
	}, nil
}
