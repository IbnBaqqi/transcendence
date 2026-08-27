package app

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/handler"
	mw "github.com/IbnBaqqi/transcendence/internal/middleware"
	"github.com/IbnBaqqi/transcendence/internal/oauth"
	"github.com/IbnBaqqi/transcendence/internal/presence"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

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
		appService.Profile,
		appService.Follow,
		appService.Block,
		appService.APIKey,
		appService.ListingImage,
		appService.Report,
		appService.Upload.MaxBytes,
		appService.AuthConfig.CookieSecure,
		oauth.NewRegistry(appService.AuthConfig),
		appService.AuthConfig.FrontendURL,
	)

	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	// r.Use(middleware.ClientIPFromHeader("X-Real-IP")) we switch to this after nginx proxy is setup
	r.Use(mw.Logger(log))
	r.Use(mw.Recoverer(log))

	authenticate := mw.Authenticate(appService.JWT, appService.APIKey)

	r.Get("/health", h.Health)

	uploads := uploadFileServer(appService.Files.Dir())
	r.Get(dtos.UploadURLPrefix+"*", uploads.ServeHTTP)
	r.Head(dtos.UploadURLPrefix+"*", uploads.ServeHTTP)

	mountDocs(r)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authenticate)
		r.Use(mw.TouchLastSeen(appService.DB.Queries, presence.Interval))
		r.Use(mw.RateLimitByKey(appService.AuthConfig.RateLimitPerMinute))
		r.Use(mw.TouchAPIKey(appService.DB.Queries, presence.Interval))

		r.Post("/auth/signup", h.Signup)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/logout", h.Logout)
		r.Post("/auth/refresh", h.Refresh)

		r.Get("/auth/oauth/{provider}", h.OAuthStart)
		r.Get("/auth/oauth/{provider}/callback", h.OAuthCallback)

		r.Get("/listings", h.GetListings)
		r.Get("/listings/search", h.SearchListings)
		r.Get("/listings/{id}", h.GetListing)
		r.Get("/listings/{id}/images", h.GetListingImages)

		r.Get("/users/{id}", h.GetPublicProfile)

		r.Group(func(r chi.Router) {
			r.Use(mw.RequiredAuth)

			r.Post("/listings", h.CreateListing)
			r.Put("/listings/{id}", h.UpdateListing)
			r.Delete("/listings/{id}", h.DeleteListing)
			r.Post("/listings/{id}/save", h.SaveListing)
			r.Delete("/listings/{id}/save", h.UnsaveListing)
			r.Get("/me/saved", h.GetSavedListings)
			r.Post("/listings/{id}/images", h.UploadListingImage)
			r.Delete("/listings/{id}/images/{imageID}", h.DeleteListingImage)
			r.Post("/listings/{id}/report", h.ReportListing)

			r.Post("/conversations", h.StartConversation)
			r.Get("/conversations", h.GetConversations)
			r.Get("/conversations/{id}", h.GetConversation)
			r.Post("/conversations/{id}/accept", h.AcceptConversation)
			r.Post("/conversations/{id}/decline", h.DeclineConversation)
			r.Get("/conversations/{id}/messages", h.GetMessages)
			r.Post("/conversations/{id}/messages", h.SendMessage)
			r.Post("/conversations/{id}/read", h.MarkConversationRead)

			r.Get("/auth/me", h.Me)

			r.Group(func(r chi.Router) {
				r.Use(mw.SessionOnly)

				r.Post("/me/api-keys", h.CreateAPIKey)
				r.Get("/me/api-keys", h.GetAPIKeys)
				r.Delete("/me/api-keys/{id}", h.RevokeAPIKey)
			})

			r.Get("/me/settings", h.GetSettings)
			r.Patch("/me/settings", h.UpdateSettings)
			r.Get("/me/unread", h.GetUnreadCount)

			r.Get("/me/profile", h.GetOwnProfile)
			r.Patch("/me/profile", h.UpdateOwnProfile)
			r.Post("/me/avatar", h.UploadAvatar)
			r.Delete("/me/avatar", h.DeleteAvatar)
			r.Post("/users/{id}/follow", h.FollowUser)
			r.Delete("/users/{id}/follow", h.UnfollowUser)
			r.Get("/users/{id}/followers", h.GetFollowers)
			r.Get("/users/{id}/following", h.GetUserFollowing)
			r.Get("/me/following", h.GetFollowing)
			r.Post("/users/{id}/block", h.BlockUser)
			r.Delete("/users/{id}/block", h.UnblockUser)
			r.Get("/me/blocks", h.GetBlocks)

			r.Post("/orders", h.CreateOrder)
			r.Get("/orders", h.GetOrders)
			r.Get("/orders/{id}", h.GetOrder)
			r.Get("/orders/{id}/events", h.GetOrderEvents)

			r.Post("/orders/{id}/confirm", h.ConfirmOrder)
			r.Post("/orders/{id}/handover", h.HandoverOrder)
			r.Post("/orders/{id}/receive", h.ReceiveOrder)
			r.Post("/orders/{id}/cancel", h.CancelOrder)
			// r.Get("/dashboard", dashboardHandler)
		})
	})

	return r
}

func uploadFileServer(dir string) http.Handler {
	fs := http.StripPrefix(dtos.UploadURLPrefix, http.FileServer(http.Dir(dir)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, dtos.UploadURLPrefix)
		if name == "" || strings.Contains(name, "/") {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")

		fs.ServeHTTP(w, r)
	})
}
