package store

import (
	"context"
	"database/sql"

	"github.com/url-shortener/internal/service"
)

// Store wraps the sqlc-generated Queries to provide a clean interface
// for the service layer.
type Store struct {
	q *Queries
}

// NewStore creates a new Store backed by the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{q: New(db)}
}

// InsertURL inserts a new URL mapping. Returns an error if the slug already exists
// (UNIQUE constraint violation).
func (s *Store) InsertURL(ctx context.Context, slug, originalURL string) error {
	_, err := s.q.InsertUrl(ctx, InsertUrlParams{
		Slug:        slug,
		OriginalUrl: originalURL,
	})
	return err
}

// GetBySlug retrieves the original URL for the given slug.
// Returns sql.ErrNoRows if the slug is not found.
func (s *Store) GetBySlug(ctx context.Context, slug string) (string, error) {
	return s.q.GetBySlug(ctx, slug)
}

// IncrementClicks atomically increments the click counter for the given slug.
func (s *Store) IncrementClicks(ctx context.Context, slug string) error {
	return s.q.IncrementClicks(ctx, slug)
}

// GetStats retrieves statistics for the given slug.
// Returns (nil, sql.ErrNoRows) if the slug is not found.
func (s *Store) GetStats(ctx context.Context, slug string) (*service.URLStats, error) {
	row, err := s.q.GetStats(ctx, slug)
	if err != nil {
		return nil, err
	}
	return &service.URLStats{
		OriginalURL: row.OriginalUrl,
		ClickCount:  row.ClickCount,
		CreatedAt:   row.CreatedAt,
	}, nil
}

// ListSlugs returns all slugs ordered by creation date (newest first).
func (s *Store) ListSlugs(ctx context.Context) ([]service.SlugEntry, error) {
	rows, err := s.q.ListSlugs(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]service.SlugEntry, len(rows))
	for i, row := range rows {
		entries[i] = service.SlugEntry{
			Slug:        row.Slug,
			OriginalURL: row.OriginalUrl,
			ClickCount:  row.ClickCount,
			CreatedAt:   row.CreatedAt,
		}
	}
	return entries, nil
}