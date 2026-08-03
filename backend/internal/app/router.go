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

	// Create handlers with injected dependencies
	h := handler.New(
		appService.DB,
		appService.Auth,
		appService.Listing,
		appService.Saved,
	)

	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	// r.Use(middleware.ClientIPFromHeader("X-Real-IP")) we switch to this after nginx proxy is setup
	r.Use(mw.Logger(log))
	r.Use(mw.Recoverer(log))

	authenticate := mw.Authenticate(appService.JWT)

	r.Get("/health", h.Health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/signup", h.Signup)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/logout", h.Logout)

		r.Group(func(r chi.Router) {
			r.Use(authenticate) // checks token, sets user in context

			r.Get("/listings", h.GetListings)
			r.Post("/listings", h.CreateListing)
			r.Get("/listings/{id}", h.GetListing)
			r.Put("/listings/{id}", h.UpdateListing)
			r.Delete("/listings/{id}", h.DeleteListing)
			r.Post("/listings/{id}/save", h.SaveListing)
			r.Delete("/listings/{id}/save", h.UnsaveListing)
			r.Get("/me/saved", h.GetSavedListings)

			// r.Get("/dashboard", dashboardHandler)
			// r.Get("/profile", profileHandler)
		})
	})

	return r
}
