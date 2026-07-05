package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
)

const slugChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const slugLength = 5
const maxCollisionRetries = 5

var (
	ErrInvalidURL   = errors.New("invalid URL: must be a valid HTTP or HTTPS URL")
	ErrURLRequired  = errors.New("url is required")
	ErrSlugNotFound = errors.New("slug not found")
)

// Service implements the business logic for the URL shortener.
type Service struct {
	store   Store
	baseURL string
}

// New creates a new Service with the given store and base URL.
func New(store Store, baseURL string) *Service {
	return &Service{
		store:   store,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// CreateResponse is returned by CreateShortURL on success.
type CreateResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
}

// CreateShortURL validates a URL, generates a unique slug, and persists the mapping.
func (s *Service) CreateShortURL(ctx context.Context, rawURL string) (*CreateResponse, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, ErrURLRequired
	}

	if !isValidURL(rawURL) {
		return nil, ErrInvalidURL
	}

	var slug string
	var lastErr error

	for attempt := 0; attempt < maxCollisionRetries; attempt++ {
		var err error
		slug, err = generateSlug()
		if err != nil {
			return nil, fmt.Errorf("failed to generate slug: %w", err)
		}

		err = s.store.InsertURL(ctx, slug, rawURL)
		if err == nil {
			// Success — slug was unique
			return &CreateResponse{
				ShortCode: slug,
				ShortURL:  s.baseURL + "/" + slug,
			}, nil
		}

		// Check if it's a UNIQUE constraint violation (collision)
		if isCollisionError(err) {
			lastErr = err
			continue
		}

		// Some other error — fail immediately
		return nil, fmt.Errorf("failed to insert URL: %w", err)
	}

	return nil, fmt.Errorf("failed to generate unique slug after %d retries: %w", maxCollisionRetries, lastErr)
}

// RedirectBySlug looks up a slug, increments its click count, and returns the original URL.
// Returns ErrSlugNotFound if the slug doesn't exist.
func (s *Service) RedirectBySlug(ctx context.Context, slug string) (string, error) {
	originalURL, err := s.store.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrSlugNotFound
		}
		return "", fmt.Errorf("failed to look up slug: %w", err)
	}

	// Increment click count — best effort: don't fail the redirect if tracking fails
	_ = s.store.IncrementClicks(ctx, slug)

	return originalURL, nil
}

// GetStats returns usage statistics for the given slug.
// Returns ErrSlugNotFound if the slug doesn't exist.
func (s *Service) GetStats(ctx context.Context, slug string) (*URLStats, error) {
	stats, err := s.store.GetStats(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSlugNotFound
		}
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	return stats, nil
}

// ListSlugsPaginated returns a paginated list of slugs ordered by creation date (newest first).
func (s *Service) ListSlugsPaginated(ctx context.Context, page, limit int) (*PaginatedSlugs, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}

	total, err := s.store.CountSlugs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count slugs: %w", err)
	}

	totalPages := int(total / int64(limit))
	if int(total)%limit != 0 {
		totalPages++
	}
	if totalPages > 0 && page > totalPages {
		page = totalPages
	} else if totalPages == 0 {
		page = 1
	}

	offset := (page - 1) * limit

	entries, err := s.store.ListSlugsPaginated(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list slugs: %w", err)
	}

	return &PaginatedSlugs{
		Data: entries,
		Pagination: PaginationMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// generateSlug creates a random 5-character alphanumeric string using crypto/rand.
func generateSlug() (string, error) {
	slug := make([]byte, slugLength)
	for i := 0; i < slugLength; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(slugChars))))
		if err != nil {
			return "", err
		}
		slug[i] = slugChars[n.Int64()]
	}
	return string(slug), nil
}

// isValidURL checks that the raw URL is a valid HTTP or HTTPS URL.
func isValidURL(rawURL string) bool {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// isCollisionError checks if an insert error is due to a UNIQUE constraint violation.
func isCollisionError(err error) bool {
	// SQLite UNIQUE constraint violation error messages contain this string
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
