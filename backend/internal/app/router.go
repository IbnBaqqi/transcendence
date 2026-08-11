package app

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/handler"
	mw "github.com/IbnBaqqi/transcendence/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter takes *database.Queries so it can construct the listing handler
func NewRouter(log *slog.Logger, appService *api) http.Handler {
	r := chi.NewRouter()

	h := handler.New(
		appService.DB,
		appService.Auth,
		appService.Listing,
		appService.Order,
		appService.Saved,
		appService.ListingImage,
		appService.Upload.MaxBytes,
	)

	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	// r.Use(middleware.ClientIPFromHeader("X-Real-IP")) we switch to this after nginx proxy is setup
	r.Use(mw.Logger(log))
	r.Use(mw.Recoverer(log))

	authenticate := mw.Authenticate(appService.JWT)

	r.Get("/health", h.Health)

	r.Handle(dtos.UploadURLPrefix+"*", uploadFileServer(appService.Files.Dir()))

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authenticate)

		r.Post("/auth/signup", h.Signup)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/logout", h.Logout)

		r.Get("/listings", h.GetListings)
		r.Get("/listings/{id}", h.GetListing)
		r.Get("/listings/{id}/images", h.GetListingImages)

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

// uploadFileServer serves stored files by bare filename, with no directory listing.
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
