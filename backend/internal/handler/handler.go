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
	Report         *service.ReportService
	Moderation     *service.ModerationService
	maxUploadBytes int64
	cookieSecure   bool
	oauth          *oauth.Registry
	frontendURL    string
}

// Deps is everything a Handler needs, by name.
//
// This was eighteen positional parameters, fourteen of them
// *service.Something. Transposing two compiled cleanly and produced a handler
// that called the wrong service - and `go build` could not catch it, because
// it does not compile tests. Named fields make that a compile error, and let a
// test set the two dependencies it exercises and omit the rest.
type Deps struct {
	DB           *database.DB
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

	MaxUploadBytes int64
	CookieSecure   bool
	OAuth          *oauth.Registry
	FrontendURL    string
}

func New(d Deps) *Handler {
	return &Handler{
		db:             d.DB,
		Auth:           d.Auth,
		Listing:        d.Listing,
		Order:          d.Order,
		Saved:          d.Saved,
		Conversation:   d.Conversation,
		User:           d.User,
		Profile:        d.Profile,
		Follow:         d.Follow,
		Block:          d.Block,
		APIKey:         d.APIKey,
		ListingImage:   d.ListingImage,
		Report:         d.Report,
		Moderation:     d.Moderation,
		maxUploadBytes: d.MaxUploadBytes,
		cookieSecure:   d.CookieSecure,
		oauth:          d.OAuth,
		frontendURL:    d.FrontendURL,
	}
}
