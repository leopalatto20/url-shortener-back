package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockStore is a testify mock implementing the Store interface.
type mockStore struct {
	mock.Mock
}

func (m *mockStore) InsertURL(ctx context.Context, slug, originalURL string) error {
	args := m.Called(ctx, slug, originalURL)
	return args.Error(0)
}

func (m *mockStore) GetBySlug(ctx context.Context, slug string) (string, error) {
	args := m.Called(ctx, slug)
	return args.String(0), args.Error(1)
}

func (m *mockStore) IncrementClicks(ctx context.Context, slug string) error {
	args := m.Called(ctx, slug)
	return args.Error(0)
}

func (m *mockStore) GetStats(ctx context.Context, slug string) (*URLStats, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*URLStats), args.Error(1)
}

func (m *mockStore) ListSlugs(ctx context.Context) ([]SlugEntry, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]SlugEntry), args.Error(1)
}

func TestService_CreateShortURL_Success(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	store.On("InsertURL", mock.Anything, mock.AnythingOfType("string"), "https://example.com").
		Return(nil).
		Once()

	resp, err := svc.CreateShortURL(context.Background(), "https://example.com")
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Len(t, resp.ShortCode, 5)
	assert.Contains(t, resp.ShortURL, resp.ShortCode)
	assert.Contains(t, resp.ShortURL, "http://short.local")

	// Verify slug contains only alphanumeric characters
	for _, c := range resp.ShortCode {
		assert.True(t, (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'),
			"slug contains non-alphanumeric character: %c", c)
	}

	store.AssertExpectations(t)
}

func TestService_CreateShortURL_InvalidURL(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	testCases := []struct {
		name string
		url  string
	}{
		{"not a url", "not-a-url"},
		{"ftp scheme", "ftp://example.com"},
		{"no scheme", "example.com"},
		{"mailto", "mailto:user@example.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateShortURL(context.Background(), tc.url)
			assert.ErrorIs(t, err, ErrInvalidURL)
		})
	}

	store.AssertExpectations(t)
}

func TestService_CreateShortURL_EmptyBody(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	_, err := svc.CreateShortURL(context.Background(), "")
	assert.ErrorIs(t, err, ErrURLRequired)

	_, err = svc.CreateShortURL(context.Background(), "   ")
	assert.ErrorIs(t, err, ErrURLRequired)

	store.AssertExpectations(t)
}

func TestService_CreateShortURL_CollisionRetry(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	// First two attempts collide; third succeeds
	store.On("InsertURL", mock.Anything, mock.AnythingOfType("string"), "https://example.com").
		Return(errors.New("UNIQUE constraint failed: urls.slug")).
		Times(2)

	store.On("InsertURL", mock.Anything, mock.AnythingOfType("string"), "https://example.com").
		Return(nil).
		Once()

	resp, err := svc.CreateShortURL(context.Background(), "https://example.com")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.ShortCode, 5)

	store.AssertExpectations(t)
}

func TestService_CreateShortURL_CollisionExhausted(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	// All attempts collide
	store.On("InsertURL", mock.Anything, mock.AnythingOfType("string"), "https://example.com").
		Return(errors.New("UNIQUE constraint failed: urls.slug")).
		Times(maxCollisionRetries)

	_, err := svc.CreateShortURL(context.Background(), "https://example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate unique slug")

	store.AssertExpectations(t)
}

func TestService_RedirectBySlug_Found(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	store.On("GetBySlug", mock.Anything, "abc12").
		Return("https://example.com", nil).
		Once()

	store.On("IncrementClicks", mock.Anything, "abc12").
		Return(nil).
		Once()

	url, err := svc.RedirectBySlug(context.Background(), "abc12")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", url)

	store.AssertExpectations(t)
}

func TestService_RedirectBySlug_NotFound(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	store.On("GetBySlug", mock.Anything, "nonexistent").
		Return("", sql.ErrNoRows).
		Once()

	_, err := svc.RedirectBySlug(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrSlugNotFound)

	store.AssertExpectations(t)
}

func TestService_RedirectBySlug_NoClickOn404(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	store.On("GetBySlug", mock.Anything, "nonexistent").
		Return("", sql.ErrNoRows).
		Once()

	// IncrementClicks should NOT be called
	_, err := svc.RedirectBySlug(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrSlugNotFound)

	store.AssertExpectations(t)
}

func TestService_RedirectBySlug_ClickIncremented(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	store.On("GetBySlug", mock.Anything, "abc12").
		Return("https://example.com", nil).
		Once()

	store.On("IncrementClicks", mock.Anything, "abc12").
		Return(nil).
		Once()

	_, err := svc.RedirectBySlug(context.Background(), "abc12")
	require.NoError(t, err)

	store.AssertExpectations(t)
}

func TestService_GetStats_Found(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	expected := &URLStats{
		OriginalURL: "https://example.com",
		ClickCount:  5,
	}

	store.On("GetStats", mock.Anything, "abc12").
		Return(expected, nil).
		Once()

	stats, err := svc.GetStats(context.Background(), "abc12")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", stats.OriginalURL)
	assert.Equal(t, int64(5), stats.ClickCount)

	store.AssertExpectations(t)
}

func TestService_GetStats_NotFound(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	store.On("GetStats", mock.Anything, "nonexistent").
		Return(nil, sql.ErrNoRows).
		Once()

	_, err := svc.GetStats(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrSlugNotFound)

	store.AssertExpectations(t)
}

func TestService_ListSlugs_Success(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	expected := []SlugEntry{
		{Slug: "xyz99", OriginalURL: "https://newest.com", ClickCount: 0},
		{Slug: "abc12", OriginalURL: "https://oldest.com", ClickCount: 5},
	}
	store.On("ListSlugs", mock.Anything).Return(expected, nil).Once()

	entries, err := svc.ListSlugs(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expected, entries)

	store.AssertExpectations(t)
}

func TestService_ListSlugs_StoreError(t *testing.T) {
	store := new(mockStore)
	svc := New(store, "http://short.local")

	store.On("ListSlugs", mock.Anything).Return(nil, assert.AnError).Once()

	_, err := svc.ListSlugs(context.Background())
	assert.Error(t, err)

	store.AssertExpectations(t)
}

func TestService_GenerateSlug_LengthAndChars(t *testing.T) {
	for i := 0; i < 20; i++ {
		slug, err := generateSlug()
		require.NoError(t, err)
		assert.Len(t, slug, 5)

		for _, c := range slug {
			assert.True(t, (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'),
				"slug contains non-alphanumeric character: %c", c)
		}
	}
}

func TestService_IsValidURL(t *testing.T) {
	assert.True(t, isValidURL("https://example.com"))
	assert.True(t, isValidURL("http://example.com/path?q=1"))
	assert.True(t, isValidURL("https://example.com:8080/path"))
	assert.False(t, isValidURL("not-a-url"))
	assert.False(t, isValidURL(""))
	assert.False(t, isValidURL("ftp://example.com"))
	assert.False(t, isValidURL("javascript:alert(1)"))
}
