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
		appService.Order,
	)

	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	// r.Use(middleware.ClientIPFromHeader("X-Real-IP")) we switch to this after nginx proxy is setup
	r.Use(mw.Logger(log))
	r.Use(mw.Recoverer(log))

	authenticate := mw.Authenticate(appService.JWT)

	r.Get("/health", h.Health)

	r.Route("/api/v1", func(r chi.Router) {
		// Optional auth for EVERY route below: attaches the user when a valid
		// token is present, does nothing when it isn't. It never rejects -
		// that's RequiredAuth's job.
		r.Use(authenticate)

		// --- Public: no token required ---
		r.Post("/auth/signup", h.Signup)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/logout", h.Logout)

		// Browsing is open so people can look around before signing up - the
		// "browse anonymously, log in to buy" flow.
		r.Get("/listings", h.GetListings)
		r.Get("/listings/{id}", h.GetListing)

		// --- Protected: 401 without a valid token ---
		// Note GET /listings/{id} is public above while PUT/DELETE on the same
		// path are in here. chi keys handlers by method, so that's fine.
		r.Group(func(r chi.Router) {
			r.Use(mw.RequiredAuth) // checks token, sets user in context

			r.Post("/listings", h.CreateListing)
			r.Put("/listings/{id}", h.UpdateListing)
			r.Delete("/listings/{id}", h.DeleteListing)

			r.Post("/orders", h.CreateOrder)
			r.Get("/orders", h.GetOrders)
			r.Get("/orders/{id}", h.GetOrder)

			// Completion is a two-sided handshake: the seller marks the
			// handover, the buyer marks receipt, and the order only becomes
			// "completed" once both have. Payment happens between them
			// off-platform, so there's no pay step.
			r.Post("/orders/{id}/confirm", h.ConfirmOrder)
			r.Post("/orders/{id}/handover", h.HandoverOrder)
			r.Post("/orders/{id}/receive", h.ReceiveOrder)
			r.Post("/orders/{id}/cancel", h.CancelOrder)
			// r.Get("/dashboard", dashboardHandler)
			// r.Get("/profile", profileHandler)
		})
	})

	return r
}
