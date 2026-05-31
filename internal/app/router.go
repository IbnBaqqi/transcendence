package app

import (
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/handler"
	mw "github.com/IbnBaqqi/transcendence/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	)

func NewRouter(log *slog.Logger) http.Handler {
r := chi.NewRouter()
	
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mw.Logger(log))
	r.Use(mw.Recoverer(log))

	healthHandler := handler.NewHealthHandler()
	r.Get("/health", healthHandler.Check)

	return r
	}
