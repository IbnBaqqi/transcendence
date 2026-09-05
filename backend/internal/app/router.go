package app

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/handler"
	mw "github.com/IbnBaqqi/transcendence/internal/middleware"
	"github.com/IbnBaqqi/transcendence/internal/oauth"
	"github.com/IbnBaqqi/transcendence/internal/presence"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// newHandler is separate from NewRouter so a test can build it and check that
// every dependency actually arrived - see TestEveryHandlerDependencyIsWired.
// The Deps struct made a forgotten field compile, so the check moved here.
func newHandler(appService *api) *handler.Handler {
	return handler.New(handler.Deps{
		DB:           appService.DB,
		Auth:         appService.Auth,
		Listing:      appService.Listing,
		Order:        appService.Order,
		Saved:        appService.Saved,
		Conversation: appService.Conversation,
		User:         appService.User,
		Profile:      appService.Profile,
		Follow:       appService.Follow,
		Notification: appService.Notification,
		Block:        appService.Block,
		APIKey:       appService.APIKey,
		ListingImage: appService.ListingImage,
		Report:       appService.Report,
		Moderation:   appService.Moderation,
		AdminOrder:   appService.AdminOrder,
		AdminUser:    appService.AdminUser,
		Review:       appService.Review,

		MaxUploadBytes: appService.Upload.MaxBytes,
		CookieSecure:   appService.AuthConfig.CookieSecure,
		OAuth:          oauth.NewRegistry(appService.AuthConfig),
		FrontendURL:    appService.AuthConfig.FrontendURL,
	})
}

func NewRouter(log *slog.Logger, appService *api) http.Handler {
	r := chi.NewRouter()

	h := newHandler(appService)

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
		// Derived from the upload cap, never a literal: this wraps r.Body
		// first, so it is the ceiling for uploads too, and a hardcoded number
		// would break them the day MAX_UPLOAD_BYTES is raised past it. The
		// headroom keeps upload.go's own cap the one that fires for uploads,
		// so their 413 still says which image was too large.
		//
		// First on purpose, ahead of the three below: TouchLastSeen writes to
		// the database, and an oversize request has no business reaching a
		// write. The trade is that such requests never reach the rate limiter
		// and so are not counted against it - acceptable, because refusing one
		// is a header parse and an integer comparison, which costs about what
		// an unrouted path costs.
		r.Use(mw.MaxBody(appService.Upload.MaxBytes + uploadHeadroom))
		r.Use(authenticate)
		r.Use(mw.TouchLastSeen(appService.DB.Queries, presence.Interval))
		r.Use(mw.RateLimitByKey(appService.AuthConfig.RateLimitPerMinute))
		r.Use(mw.TouchAPIKey(appService.DB.Queries, presence.Interval))

		r.Post("/auth/signup", h.Signup)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/logout", h.Logout)
		r.Post("/auth/refresh", h.Refresh)
		r.Post("/auth/forgot-password", h.ForgotPassword)
		r.Post("/auth/reset-password", h.ResetPassword)

		r.Get("/auth/providers", h.OAuthProviders)
		r.Get("/auth/oauth/{provider}", h.OAuthStart)
		r.Get("/auth/oauth/{provider}/callback", h.OAuthCallback)

		r.Get("/listings/search", h.SearchListings)
		r.Get("/listings/{id}", h.GetListing)
		r.Get("/listings/{id}/images", h.GetListingImages)

		r.Get("/categories", h.GetCategories)

		r.Get("/users/{id}", h.GetPublicProfile)
		r.Get("/users/{id}/reviews", h.GetSellerReviews)

		r.Group(func(r chi.Router) {
			r.Use(mw.RequiredAuth)
			r.Use(mw.RequireActiveUser(appService.DB.Queries))

			r.Post("/listings", h.CreateListing)
			r.Put("/listings/{id}", h.UpdateListing)
			r.Delete("/listings/{id}", h.DeleteListing)
			r.Post("/listings/{id}/save", h.SaveListing)
			r.Delete("/listings/{id}/save", h.UnsaveListing)
			r.Get("/me/saved", h.GetSavedListings)
			r.Post("/listings/{id}/images", h.UploadListingImage)
			r.Delete("/listings/{id}/images/{imageID}", h.DeleteListingImage)
			r.Put("/listings/{id}/images/order", h.ReorderListingImages)
			r.Post("/listings/{id}/report", h.ReportListing)

			r.Group(func(r chi.Router) {
				r.Use(mw.RequireRole(appService.DB.Queries, auth.RoleAdmin))

				r.Get("/admin/reports", h.GetReportQueue)
				r.Get("/admin/listings/{id}/reports", h.GetListingReports)
				r.Get("/admin/listings/{id}/moderation", h.GetModerationHistory)
				r.Post("/admin/listings/{id}/moderate", h.ModerateListing)

				r.Get("/admin/orders", h.ListOrdersForAdmin)
				r.Post("/admin/orders/{id}/resolve", h.ResolveOrder)
				r.Get("/admin/orders/{id}/events", h.GetOrderEventsForAdmin)
				r.Get("/admin/users", h.ListUsers)
				r.Get("/admin/users/{id}/history", h.GetUserHistory)
				r.Post("/admin/users/{id}/suspend", h.SuspendUser)
				r.Post("/admin/users/{id}/reinstate", h.ReinstateUser)
				r.Patch("/admin/users/{id}/role", h.SetUserRole)
				r.Delete("/admin/users/{id}", h.DeleteUserAsAdmin)
			})

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

				// SessionOnly, like account deletion beside it: an endpoint that
				// answers with everything about a person in one response is
				// exactly what a leaked API key must not be able to call.
				r.With(mw.RateLimitByUser(exportsPerHour)).
					Get("/me/export", h.ExportMyData)

				r.Delete("/me", h.DeleteAccount)
				r.Post("/me/password", h.ChangePassword)
			})

			r.Get("/me/settings", h.GetSettings)
			r.Patch("/me/settings", h.UpdateSettings)
			r.Get("/me/unread", h.GetUnreadCount)

			r.Get("/me/notifications", h.GetNotifications)
			r.Post("/me/notifications/read", h.MarkNotificationsRead)

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

			r.Get("/orders/{id}/review", h.GetOrderReview)
			r.Post("/orders/{id}/review", h.CreateReview)
			r.Patch("/reviews/{id}", h.UpdateReview)
			// r.Get("/dashboard", dashboardHandler)
		})
	})

	return r
}

// Room above the upload cap for the multipart envelope around a file, so a
// legal upload meets upload.go's limit rather than this one.
const uploadHeadroom = 1 << 20

const exportsPerHour = 3

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
