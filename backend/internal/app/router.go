package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/handler"
	mw "github.com/IbnBaqqi/transcendence/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const presenceInterval = time.Minute

// NewRouter takes *database.Queries so it can construct the listing handler
func NewRouter(log *slog.Logger, appService *api) http.Handler {
	r := chi.NewRouter()

	h := handler.New(
		appService.DB,
		appService.Auth,
		appService.Listing,
		appService.Order,
		appService.Saved,
		appService.Conversation,
		appService.User,
	)

	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	// r.Use(middleware.ClientIPFromHeader("X-Real-IP")) we switch to this after nginx proxy is setup
	r.Use(mw.Logger(log))
	r.Use(mw.Recoverer(log))

	authenticate := mw.Authenticate(appService.JWT)

	r.Get("/health", h.Health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authenticate)
		r.Use(mw.TouchLastSeen(appService.DB.Queries, presenceInterval))

		r.Post("/auth/signup", h.Signup)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/logout", h.Logout)

		r.Get("/listings", h.GetListings)
		r.Get("/listings/{id}", h.GetListing)

		r.Group(func(r chi.Router) {
			r.Use(mw.RequiredAuth)

			r.Post("/listings", h.CreateListing)
			r.Put("/listings/{id}", h.UpdateListing)
			r.Delete("/listings/{id}", h.DeleteListing)
			r.Post("/listings/{id}/save", h.SaveListing)
			r.Delete("/listings/{id}/save", h.UnsaveListing)
			r.Get("/me/saved", h.GetSavedListings)

			r.Post("/conversations", h.StartConversation)
			r.Get("/conversations", h.GetConversations)
			r.Get("/conversations/{id}", h.GetConversation)
			r.Post("/conversations/{id}/accept", h.AcceptConversation)
			r.Post("/conversations/{id}/decline", h.DeclineConversation)
			r.Get("/conversations/{id}/messages", h.GetMessages)
			r.Post("/conversations/{id}/messages", h.SendMessage)
			r.Post("/conversations/{id}/read", h.MarkConversationRead)

			r.Get("/me/settings", h.GetSettings)
			r.Patch("/me/settings", h.UpdateSettings)
			r.Get("/me/unread", h.GetUnreadCount)

			r.Post("/orders", h.CreateOrder)
			r.Get("/orders", h.GetOrders)
			r.Get("/orders/{id}", h.GetOrder)

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
