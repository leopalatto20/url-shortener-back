package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/url-shortener/internal/service"
)

// ShortenRequest is the JSON body for POST /shorten.
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse is the JSON body for a successful 201 response.
type ShortenResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
}

// StatsResponse is the JSON body for GET /:slug/stats.
type StatsResponse struct {
	OriginalURL string `json:"original_url"`
	ClickCount  int64  `json:"click_count"`
	CreatedAt   string `json:"created_at"`
}

// errorResponse is sent back on validation/not-found errors.
type errorResponse struct {
	Error string `json:"error"`
}

// Handler wires HTTP handlers to service operations.
type Handler struct {
	svc *service.Service
}

// New creates a new Handler.
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// HandleShorten handles POST /shorten — validates input, creates short URL, returns 201.
func (h *Handler) HandleShorten(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "request body is required"})
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	resp, err := h.svc.CreateShortURL(r.Context(), req.URL)
	if err != nil {
		if errors.Is(err, service.ErrURLRequired) || errors.Is(err, service.ErrInvalidURL) {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusCreated, ShortenResponse{
		ShortCode: resp.ShortCode,
		ShortURL:  resp.ShortURL,
	})
}

// HandleRedirect handles GET /:slug — resolves slug and redirects or returns 404.
func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	originalURL, err := h.svc.RedirectBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrSlugNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "short URL not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}

// HandleStats handles GET /:slug/stats — returns stats or 404.
func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	stats, err := h.svc.GetStats(r.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrSlugNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "short URL not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, StatsResponse{
		OriginalURL: stats.OriginalURL,
		ClickCount:  stats.ClickCount,
		CreatedAt:   stats.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// writeJSON is a helper to write a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}