package app

import (
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/handler"
	mw "github.com/IbnBaqqi/transcendence/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	)

// TODO(listings): NewRouter currently only takes a logger, but listing
// routes need a *database.Queries (or *database.DB, which embeds it)
// to construct ListingHandler. This will require:
//   1. Changing this signature to NewRouter(log *slog.Logger, db *database.DB)
//   2. Uncommenting/finishing app.New(cfg, db) in main.go, which already
//      exists but is commented out
//   3. Passing db through from main.go into NewRouter(...)
// Not doing this yet — flagging for team discussion since it touches
// shared scaffold files (app.go, main.go), not just listing-specific code.
func NewRouter(log *slog.Logger) http.Handler {
r := chi.NewRouter()
	
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mw.Logger(log))
	r.Use(mw.Recoverer(log))

	healthHandler := handler.NewHealthHandler()
	r.Get("/health", healthHandler.Check)

	// TODO(listings): once db is available here, wire up:
	// listingHandler := handler.NewListingHandler(db.Queries)
	// r.Get("/listings", listingHandler.List)
	// r.Post("/listings", listingHandler.Create)
	// r.Get("/listings/{id}", listingHandler.Get)
	// r.Put("/listings/{id}", listingHandler.Update)
	// r.Delete("/listings/{id}", listingHandler.Delete)

	return r
	}
