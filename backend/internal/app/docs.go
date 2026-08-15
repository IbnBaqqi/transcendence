package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	swgui "github.com/swaggest/swgui/v5emb"

	// Aliased: this package already has a type called api (app.go).
	openapi "github.com/IbnBaqqi/transcendence/api"
)

const (
	specPath = "/api/openapi.yaml"
	docsPath = "/api/docs"
)

// mountDocs serves the spec and a Swagger UI that reads it.
func mountDocs(r chi.Router) {
	r.Get(specPath, serveSpec)

	r.Mount(docsPath, swgui.NewHandler(docsTitle, specPath, docsPath))
}

// docsTitle is what the browser tab says; keep it in step with the spec's
// info.title.
const docsTitle = "Foraged goods marketplace API"

func serveSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")

	_, _ = w.Write(openapi.Spec)
}
