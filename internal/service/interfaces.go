package service

import (
	"context"
	"time"
)

// URLStats holds statistics for a shortened URL.
type URLStats struct {
	OriginalURL string    `json:"original_url"`
	ClickCount  int64     `json:"click_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// SlugEntry holds a single slug entry for the listing endpoint.
type SlugEntry struct {
	Slug        string    `json:"slug"`
	OriginalURL string    `json:"original_url"`
	ClickCount  int64     `json:"click_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// PaginationMeta holds pagination metadata for the slug listing response.
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// PaginatedSlugs is the envelope returned by ListSlugsPaginated.
type PaginatedSlugs struct {
	Data       []SlugEntry    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// Store defines the data layer contract the service depends on.
type Store interface {
	InsertURL(ctx context.Context, slug, originalURL string) error
	GetBySlug(ctx context.Context, slug string) (string, error)
	IncrementClicks(ctx context.Context, slug string) error
	GetStats(ctx context.Context, slug string) (*URLStats, error)
	ListSlugsPaginated(ctx context.Context, limit, offset int) ([]SlugEntry, error)
	CountSlugs(ctx context.Context) (int64, error)
}
