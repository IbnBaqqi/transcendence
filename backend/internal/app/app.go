// Package app wires up dependencies and defines HTTP routes for the transcendence server.
package app

import (
	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/config"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/service"
)

// api holds all dependencies for the api handlers
type api struct {
	DB      *database.DB
	JWT     *auth.JwtService
	Auth    *auth.Service
	Listing *service.ListingService
}

// New initializes all services and returns a pointer to api
func New(cfg *config.Config, db *database.DB) (*api, error) {
	jwtService := auth.NewJwtService(cfg.Auth.JWTSecret)
	authService := auth.NewService(db.Queries, jwtService)
	listingService := service.NewListingService(db.Queries)

	return &api{
		DB:      db,
		JWT:     jwtService,
		Auth:    authService,
		Listing: listingService,
	}, nil
}
