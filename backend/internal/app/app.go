// Package app wires up dependencies and defines HTTP routes for the transcendence server.
package app

import (
	"github.com/IbnBaqqi/transcendence/internal/config"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/service"
)

// api holds all dependencies for the api handlers
type api struct {
	DB      *database.DB
	Listing *service.ListingService
}

// New initializes all services and returns a pointer to api
func New(cfg *config.Config, db *database.DB) (*api, error) {

	listingService := service.NewListingService(db.Queries)

	return &api{
		DB:      db,
		Listing: listingService,
	}, nil
}
