# Design: URL Shortener Service

## Technical Approach

Three-layer architecture (Handler → Service → Store) using Go, Chi, sqlc, and SQLite. Each capability (creation, redirect, stats) flows through all three layers. sqlc generates type-safe Go from SQL; Chi handles routing; service layer owns business logic and is fully mockable via interfaces.

## Architecture Decisions

| Decision | Option A | Option B | Choice | Rationale |
|----------|----------|----------|--------|-----------|
| Project layout | Flat `internal/` packages | Nested per-capability dirs | **Flat** — `handler/`, `service/`, `store/` | Small project; per-capability dirs add indirection with no benefit |
| Slug generation | `crypto/rand` random | Hash-based (MD5/SHA prefix) | **crypto/rand** | Hashes are deterministic and leak URL similarity; random is simpler |
| HTTP router | `net/http` mux | Chi | **Chi** | Middleware ecosystem, clean param extraction, minimal overhead |
| SQL tooling | Raw `database/sql` | sqlc | **sqlc** | Type-safe queries from SQL; catches errors at compile time, not runtime |
| DB driver | `modernc.org/sqlite` (pure Go) | `mattn/go-sqlite3` (CGO) | **mattn/go-sqlite3** | Better SQLite compat; CGO acceptable for this project scope |
| Testing | Go stdlib only | testify | **testify** | Readable assertions, mock support; already in proposal deps |

## Data Flow

### POST /shorten

```
Client ──POST /shorten──→ Handler.ParseJSON()
         ↓
    Service.CreateShortURL(ctx, url)
         ↓
    Validate URL (net/url parse, scheme check)
         ↓
    Generate 5-char slug (crypto/rand)
         ↓
    Store.InsertURL(ctx, slug, url) ──→ SQLite
         ↓                              ↑ collision? retry
    Return {short_code, short_url}      │ with new slug
         ↓
    Handler → 201 JSON response
```

### GET /:slug

```
Client ──GET /:slug──→ Handler.GetSlug()
         ↓
    Service.RedirectBySlug(ctx, slug)
         ↓
    Store.GetBySlug(ctx, slug) ──→ SQLite
         ↓                           ↑ not found → 404
    Store.IncrementClicks(ctx, slug) → SQLite
         ↓
    Handler → 301 Location: original_url
```

### GET /:slug/stats

```
Client ──GET /:slug/stats──→ Handler.GetStats()
         ↓
    Service.GetStats(ctx, slug)
         ↓
    Store.GetStats(ctx, slug) ──→ SQLite
         ↓                          ↑ not found → 404
    Return {original_url, click_count, created_at}
         ↓
    Handler → 200 JSON response
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/url-shortener/main.go` | Create | Entry point: wire deps, configure Chi, start server |
| `internal/handler/handler.go` | Create | HTTP handlers for all 3 endpoints, JSON parsing/responses |
| `internal/handler/routes.go` | Create | Chi router setup, middleware, route registration |
| `internal/service/service.go` | Create | Business logic: validation, slug gen, orchestration |
| `internal/service/interfaces.go` | Create | `Store` interface for testability |
| `internal/store/store.go` | Create | sqlc-backed SQLite store implementation |
| `internal/store/querier.go` | Create | sqlc-generated Querier interface |
| `db/migrations/001_create_urls.sql` | Create | SQLite schema: `urls` table + slug index |
| `db/queries/urls.sql` | Create | sqlc query definitions (insert, get, increment, stats) |
| `sqlc.yaml` | Create | sqlc configuration pointing to db/ |
| `go.mod` | Create | Module definition with all dependencies |
| `Makefile` | Create | `build`, `test`, `run`, `sqlc-generate` targets |

## Interfaces / Contracts

```go
// internal/service/interfaces.go
type Store interface {
    InsertURL(ctx context.Context, slug, originalURL string) error
    GetBySlug(ctx context.Context, slug string) (string, error)
    IncrementClicks(ctx context.Context, slug string) error
    GetStats(ctx context.Context, slug string) (*URLStats, error)
}

type URLStats struct {
    OriginalURL string
    ClickCount  int64
    CreatedAt   time.Time
}
```

```go
// internal/handler/handler.go — request/response types
type ShortenRequest struct {
    URL string `json:"url"`
}

type ShortenResponse struct {
    ShortCode string `json:"short_code"`
    ShortURL  string `json:"short_url"`
}

type StatsResponse struct {
    OriginalURL string    `json:"original_url"`
    ClickCount  int64     `json:"click_count"`
    CreatedAt   time.Time `json:"created_at"`
}
```

### SQLite Schema

```sql
CREATE TABLE IF NOT EXISTS urls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    click_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_slug ON urls(slug);
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Service: URL validation, slug generation, collision retry, 404 handling | Mock `Store` interface; testify assertions |
| Integration | Store: insert, get, increment, stats against real SQLite | In-memory SQLite (`:memory:`); test each sqlc query |
| HTTP | Handler: full request/response cycle for all 3 endpoints + error cases | `httptest.NewRecorder` + Chi router; assert status codes and JSON bodies |

## Migration / Rollout

No migration required — greenfield project. SQLite database file created on first run.

## Open Questions

- [ ] Should `POST /shorten` deduplicate URLs (return existing slug for same URL)? Specs don't require it — MVP will create a new slug each time.
- [ ] Server listen address and port: default to `:8080`?
