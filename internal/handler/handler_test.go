package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/url-shortener/internal/service"
)

// mockStore implements the Store interface for handler-level tests.
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

func (m *mockStore) GetStats(ctx context.Context, slug string) (*service.URLStats, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.URLStats), args.Error(1)
}

func (m *mockStore) ListSlugsPaginated(ctx context.Context, limit, offset int) ([]service.SlugEntry, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]service.SlugEntry), args.Error(1)
}

func (m *mockStore) CountSlugs(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// setupTestHandler creates a router wired with a Handler → Service → mockStore chain.
func setupTestHandler(t *testing.T) (*mockStore, http.Handler) {
	t.Helper()
	ms := new(mockStore)
	svc := service.New(ms, "http://short.local")
	h := New(svc)
	router := NewRouter(h)
	return ms, router
}

func TestHandler_PostShorten_201(t *testing.T) {
	ms, router := setupTestHandler(t)

	ms.On("InsertURL", mock.Anything, mock.AnythingOfType("string"), "https://example.com").
		Return(nil).
		Once()

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var resp ShortenResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp.ShortCode, 5)
	assert.Contains(t, resp.ShortURL, resp.ShortCode)

	ms.AssertExpectations(t)
}

func TestHandler_PostShorten_400_InvalidURL(t *testing.T) {
	ms, router := setupTestHandler(t)

	body := `{"url":"not-a-url"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	err := json.NewDecoder(w.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "invalid URL")

	// No store calls should have been made
	ms.AssertNotCalled(t, "InsertURL")
}

func TestHandler_PostShorten_400_EmptyBody(t *testing.T) {
	_, router := setupTestHandler(t)

	// Empty body hits JSON decode which returns io.EOF
	req := httptest.NewRequest(http.MethodPost, "/shorten", http.NoBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	err := json.NewDecoder(w.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "invalid")
}

func TestHandler_PostShorten_400_InvalidJSON(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp errorResponse
	err := json.NewDecoder(w.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "invalid JSON")
}

func TestHandler_GetSlug_301(t *testing.T) {
	ms, router := setupTestHandler(t)

	ms.On("GetBySlug", mock.Anything, "abc12").
		Return("https://example.com", nil).
		Once()
	ms.On("IncrementClicks", mock.Anything, "abc12").
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/abc12", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Location"))

	ms.AssertExpectations(t)
}

func TestHandler_GetSlug_404(t *testing.T) {
	ms, router := setupTestHandler(t)

	ms.On("GetBySlug", mock.Anything, "nonexistent").
		Return("", sql.ErrNoRows).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp errorResponse
	err := json.NewDecoder(w.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "not found")

	// IncrementClicks should NOT be called on 404
	ms.AssertNotCalled(t, "IncrementClicks")
}

func TestHandler_GetSlugStats_200(t *testing.T) {
	ms, router := setupTestHandler(t)

	ms.On("GetStats", mock.Anything, "abc12").
		Return(&service.URLStats{
			OriginalURL: "https://example.com",
			ClickCount:  3,
		}, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/abc12/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp StatsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", resp.OriginalURL)
	assert.Equal(t, int64(3), resp.ClickCount)

	ms.AssertExpectations(t)
}

func TestHandler_GetSlugStats_404(t *testing.T) {
	ms, router := setupTestHandler(t)

	ms.On("GetStats", mock.Anything, "nonexistent").
		Return(nil, sql.ErrNoRows).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp errorResponse
	err := json.NewDecoder(w.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "not found")

	ms.AssertExpectations(t)
}

func TestHandler_ContentType_JSON(t *testing.T) {
	ms, router := setupTestHandler(t)

	// Create a URL first
	ms.On("InsertURL", mock.Anything, mock.AnythingOfType("string"), "https://example.com").
		Return(nil).
		Once()

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// We need a known slug for the stats test; extract from the response
	var createResp ShortenResponse
	err := json.NewDecoder(w.Body).Decode(&createResp)
	require.NoError(t, err)
	slug := createResp.ShortCode

	ms.On("GetStats", mock.Anything, slug).
		Return(&service.URLStats{OriginalURL: "https://example.com", ClickCount: 0}, nil).
		Once()

	req2 := httptest.NewRequest(http.MethodGet, "/"+slug+"/stats", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "application/json", w2.Header().Get("Content-Type"))

	ms.AssertExpectations(t)
}

func TestHandler_Integration_PostGetStats(t *testing.T) {
	ms, router := setupTestHandler(t)

	// POST /shorten — first insert succeeds
	ms.On("InsertURL", mock.Anything, mock.AnythingOfType("string"), "https://example.com").
		Return(nil).
		Once()

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp ShortenResponse
	err := json.NewDecoder(w.Body).Decode(&createResp)
	require.NoError(t, err)
	assert.Len(t, createResp.ShortCode, 5)
	slug := createResp.ShortCode

	// GET /:slug — redirect
	ms.On("GetBySlug", mock.Anything, slug).
		Return("https://example.com", nil).
		Once()
	ms.On("IncrementClicks", mock.Anything, slug).
		Return(nil).
		Once()

	req2 := httptest.NewRequest(http.MethodGet, "/"+slug, nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusMovedPermanently, w2.Code)

	// GET /:slug/stats — stats reflect redirect
	ms.On("GetStats", mock.Anything, slug).
		Return(&service.URLStats{
			OriginalURL: "https://example.com",
			ClickCount:  1,
		}, nil).
		Once()

	req3 := httptest.NewRequest(http.MethodGet, "/"+slug+"/stats", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	var statsResp StatsResponse
	err = json.NewDecoder(w3.Body).Decode(&statsResp)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", statsResp.OriginalURL)
	assert.Equal(t, int64(1), statsResp.ClickCount)

	ms.AssertExpectations(t)
}

func TestHandler_ListSlugs_DefaultPagination(t *testing.T) {
	ms, router := setupTestHandler(t)

	now := time.Now()
	entries := []service.SlugEntry{
		{Slug: "xyz99", OriginalURL: "https://newest.com", ClickCount: 0, CreatedAt: now},
		{Slug: "abc12", OriginalURL: "https://oldest.com", ClickCount: 5, CreatedAt: now.Add(-time.Hour)},
	}
	ms.On("CountSlugs", mock.Anything).Return(int64(2), nil).Once()
	ms.On("ListSlugsPaginated", mock.Anything, 50, 0).Return(entries, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/slugs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp SlugsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Len(t, resp.Data, 2)
	assert.Equal(t, "xyz99", resp.Data[0].Slug)
	assert.Equal(t, "https://newest.com", resp.Data[0].OriginalURL)
	assert.Equal(t, int64(0), resp.Data[0].ClickCount)
	assert.NotEmpty(t, resp.Data[0].CreatedAt)
	assert.Equal(t, "abc12", resp.Data[1].Slug)
	assert.Equal(t, 1, resp.Pagination.Page)
	assert.Equal(t, 50, resp.Pagination.Limit)
	assert.Equal(t, int64(2), resp.Pagination.Total)
	assert.Equal(t, 1, resp.Pagination.TotalPages)

	ms.AssertExpectations(t)
}

func TestHandler_ListSlugs_ExplicitPagination(t *testing.T) {
	ms, router := setupTestHandler(t)

	now := time.Now()
	entries := []service.SlugEntry{
		{Slug: "xyz99", OriginalURL: "https://newest.com", ClickCount: 0, CreatedAt: now},
	}
	ms.On("CountSlugs", mock.Anything).Return(int64(3), nil).Once()
	ms.On("ListSlugsPaginated", mock.Anything, 2, 2).Return(entries, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/slugs?page=2&limit=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp SlugsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "xyz99", resp.Data[0].Slug)
	assert.Equal(t, 2, resp.Pagination.Page)
	assert.Equal(t, 2, resp.Pagination.Limit)
	assert.Equal(t, int64(3), resp.Pagination.Total)
	assert.Equal(t, 2, resp.Pagination.TotalPages)

	ms.AssertExpectations(t)
}

func TestHandler_ListSlugs_Empty(t *testing.T) {
	ms, router := setupTestHandler(t)

	ms.On("CountSlugs", mock.Anything).Return(int64(0), nil).Once()
	ms.On("ListSlugsPaginated", mock.Anything, 50, 0).Return([]service.SlugEntry{}, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/slugs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp SlugsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Empty(t, resp.Data)
	assert.Equal(t, 0, resp.Pagination.TotalPages)

	ms.AssertExpectations(t)
}

func TestHandler_ListSlugs_Coexistence(t *testing.T) {
	ms, router := setupTestHandler(t)

	ms.On("CountSlugs", mock.Anything).Return(int64(0), nil).Once()
	ms.On("ListSlugsPaginated", mock.Anything, 50, 0).Return([]service.SlugEntry{}, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/slugs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp SlugsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	// Should be the listing, not a redirect (which would be a 301)
	assert.NotEqual(t, http.StatusMovedPermanently, w.Code)

	ms.AssertExpectations(t)
}

func TestHandler_ListSlugs_StoreError(t *testing.T) {
	ms, router := setupTestHandler(t)

	ms.On("CountSlugs", mock.Anything).Return(int64(0), assert.AnError).Once()

	req := httptest.NewRequest(http.MethodGet, "/slugs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errResp errorResponse
	err := json.NewDecoder(w.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "internal server error")

	ms.AssertExpectations(t)
}

func TestHandler_ListSlugs_InvalidQueryParams(t *testing.T) {
	ms, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/slugs?page=invalid&limit=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid page parameter", resp["error"])

	ms.AssertExpectations(t)
}

func TestHandler_ListSlugs_NegativeParams(t *testing.T) {
	ms, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/slugs?page=-5&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid page parameter", resp["error"])

	ms.AssertExpectations(t)
}

func TestHandler_ListSlugs_ZeroLimit(t *testing.T) {
	ms, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/slugs?page=1&limit=0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid limit parameter", resp["error"])

	ms.AssertExpectations(t)
}

// Chi-based direct test for routing without complex setup
func TestRoutes_AreMounted(t *testing.T) {
	ms := new(mockStore)
	svc := service.New(ms, "http://short.local")
	h := New(svc)

	// Mock InsertURL so the POST route call doesn't panic
	ms.On("InsertURL", mock.Anything, mock.AnythingOfType("string"), "https://x.com").
		Return(nil).
		Once()

	// Build chi router manually
	r := chi.NewRouter()
	r.Post("/shorten", h.HandleShorten)
	r.Get("/{slug}", h.HandleRedirect)
	r.Get("/{slug}/stats", h.HandleStats)

	// POST route exists
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":"https://x.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Will return 201 because mock is set up
	assert.NotEqual(t, http.StatusNotFound, w.Code)
	assert.NotEqual(t, http.StatusMethodNotAllowed, w.Code)

	ms.AssertExpectations(t)
}

func TestHandler_OpenAPI_Returns200(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-yaml", w.Header().Get("Content-Type"))
	assert.NotEmpty(t, w.Body.String())
}

func TestHandler_OpenAPI_BodyIsValidYAML(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var doc map[string]any
	err := yaml.Unmarshal(w.Body.Bytes(), &doc)
	require.NoError(t, err, "response body must be valid YAML")

	assert.Equal(t, "3.1.0", doc["openapi"])

	paths, ok := doc["paths"].(map[string]any)
	require.True(t, ok, "paths must be a map")
	assert.Contains(t, paths, "/shorten")
	assert.Contains(t, paths, "/{slug}")
	assert.Contains(t, paths, "/{slug}/stats")
	assert.Contains(t, paths, "/slugs")
}

func TestHandler_OpenAPI_RouteNotConflicting(t *testing.T) {
	_, router := setupTestHandler(t)

	// The literal route /openapi.yaml must NOT be caught by /{slug}
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-yaml", w.Header().Get("Content-Type"))
}
