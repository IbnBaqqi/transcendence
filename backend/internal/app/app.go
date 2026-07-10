// Package app wires up dependencies and defines HTTP routes for the transcendence server.
package app

import (
	"github.com/IbnBaqqi/transcendence/internal/config"
	"github.com/IbnBaqqi/transcendence/internal/database"
)

// API holds all dependencies for the API handlers
type API struct {
	DB *database.DB
}

// New initializes all services and returns a pointer to API
func New(cfg *config.Config, db *database.DB) (*API, error) {

	return &API{
		DB: db,
	}, nil
}
