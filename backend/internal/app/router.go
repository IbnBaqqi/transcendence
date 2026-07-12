package app

import (
	"log/slog"
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/handler"
	mw "github.com/IbnBaqqi/transcendence/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter takes *database.Queries so it can construct the listing handler
func NewRouter(log *slog.Logger, appService *api) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	// r.Use(middleware.ClientIPFromHeader("X-Real-IP")) we switch to this after nginx proxy is setup
	r.Use(mw.Logger(log))
	r.Use(mw.Recoverer(log))

	// Create handlers with injected dependencies
	h := handler.New(
		appService.DB,
		appService.Listing,
	)

	r.Get("/health", h.Health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/listings", h.GetListings)
		r.Post("/listings", h.CreateListing)
		r.Get("/listings/{id}", h.GetListing)
		r.Put("/listings/{id}", h.UpdateListing)
		r.Delete("/listings/{id}", h.DeleteListing)
	})

	return r
}
