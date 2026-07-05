package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			original_url TEXT NOT NULL,
			click_count INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_slug ON urls(slug);
	`)
	require.NoError(t, err)

	return db
}

func TestStore_InsertURL(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	err := s.InsertURL(context.Background(), "abc12", "https://example.com")
	require.NoError(t, err)

	// Duplicate slug should fail
	err = s.InsertURL(context.Background(), "abc12", "https://other.com")
	assert.Error(t, err)
}

func TestStore_GetBySlug_Found(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	err := s.InsertURL(context.Background(), "abc12", "https://example.com")
	require.NoError(t, err)

	url, err := s.GetBySlug(context.Background(), "abc12")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", url)
}

func TestStore_GetBySlug_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	_, err := s.GetBySlug(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestStore_IncrementClicks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	err := s.InsertURL(context.Background(), "abc12", "https://example.com")
	require.NoError(t, err)

	err = s.IncrementClicks(context.Background(), "abc12")
	require.NoError(t, err)

	// Verify click count increased
	stats, err := s.GetStats(context.Background(), "abc12")
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.ClickCount)
}

func TestStore_IncrementClicks_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	// Increment on non-existent slug should not error (UPDATE affects 0 rows)
	err := s.IncrementClicks(context.Background(), "nonexistent")
	assert.NoError(t, err)
}

func TestStore_GetStats_Found(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	err := s.InsertURL(context.Background(), "abc12", "https://example.com")
	require.NoError(t, err)

	stats, err := s.GetStats(context.Background(), "abc12")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", stats.OriginalURL)
	assert.Equal(t, int64(0), stats.ClickCount)
	assert.False(t, stats.CreatedAt.IsZero())
}

func TestStore_GetStats_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	_, err := s.GetStats(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestStore_GetStats_ClickCountReflectsRedirects(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	err := s.InsertURL(context.Background(), "abc12", "https://example.com")
	require.NoError(t, err)

	// Redirect 3 times
	for i := 0; i < 3; i++ {
		err := s.IncrementClicks(context.Background(), "abc12")
		require.NoError(t, err)
	}

	stats, err := s.GetStats(context.Background(), "abc12")
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.ClickCount)
}

func TestStore_NewURLHasZeroClicks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	err := s.InsertURL(context.Background(), "abc12", "https://example.com")
	require.NoError(t, err)

	stats, err := s.GetStats(context.Background(), "abc12")
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.ClickCount)
}

func TestStore_InsertAndGetBySlug_MultipleURLs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	err := s.InsertURL(context.Background(), "abc12", "https://example.com")
	require.NoError(t, err)

	err = s.InsertURL(context.Background(), "xyz99", "https://other.com")
	require.NoError(t, err)

	url1, err := s.GetBySlug(context.Background(), "abc12")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", url1)

	url2, err := s.GetBySlug(context.Background(), "xyz99")
	require.NoError(t, err)
	assert.Equal(t, "https://other.com", url2)
}

func TestStore_InsertURL_WithTimestamps(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	before := time.Now().Truncate(time.Second)
	err := s.InsertURL(context.Background(), "abc12", "https://example.com")
	require.NoError(t, err)
	after := time.Now().Add(time.Second).Truncate(time.Second)

	stats, err := s.GetStats(context.Background(), "abc12")
	require.NoError(t, err)
	assert.True(t, stats.CreatedAt.After(before) || stats.CreatedAt.Equal(before))
	assert.True(t, stats.CreatedAt.Before(after) || stats.CreatedAt.Equal(after))
}