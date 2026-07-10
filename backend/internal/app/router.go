package app

import (
	"log/slog"
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/handler"
	mw "github.com/IbnBaqqi/transcendence/internal/middleware"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter takes *database.Queries so it can construct the listing handler
func NewRouter(log *slog.Logger, queries *database.Queries) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mw.Logger(log))
	r.Use(mw.Recoverer(log))

	// health stays at the root. it's an infra endpoint, not part of the versioned API
	healthHandler := handler.NewHealthHandler()
	r.Get("/health", healthHandler.Check)

	// build the listing feature bottom-up: queries -> service -> handler
	listingSvc := service.NewListingService(queries)
	listingHandler := handler.NewListingHandler(listingSvc)

	// all API routes live under /api/v1 r.Route mounts a sub-souter.
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/listings", listingHandler.List)           // GET
		r.Post("/listings", listingHandler.Create)        // POST
		r.Get("/listings/{id}", listingHandler.Get)       // GET
		r.Put("/listings/{id}", listingHandler.Update)    // PUT
		r.Delete("/listings/{id}", listingHandler.Delete) // DELETE
	})

	return r
}
