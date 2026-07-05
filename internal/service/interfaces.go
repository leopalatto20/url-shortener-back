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

// Store defines the data layer contract the service depends on.
type Store interface {
	InsertURL(ctx context.Context, slug, originalURL string) error
	GetBySlug(ctx context.Context, slug string) (string, error)
	IncrementClicks(ctx context.Context, slug string) error
	GetStats(ctx context.Context, slug string) (*URLStats, error)
}