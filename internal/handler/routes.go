package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/url-shortener/api"
)

// NewRouter creates a Chi router with all URL shortener routes mounted.
func NewRouter(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		ExposedHeaders:   []string{"Location"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Static routes must be registered before wildcard routes
	r.Get("/openapi.yaml", serveOpenAPI)

	r.Post("/shorten", h.HandleShorten)
	r.Get("/slugs", h.HandleListSlugs)
	r.Get("/{slug}", h.HandleRedirect)
	r.Get("/{slug}/stats", h.HandleStats)

	return r
}

// serveOpenAPI serves the embedded OpenAPI 3.1.0 specification.
func serveOpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(api.OpenAPISpec) //nolint:errcheck
}
