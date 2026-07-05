## Exploration: List All Slugs

### Current State

The URL shortener uses a three-layer Go architecture (Handler → Service → Store) with Chi router and sqlc-generated SQLite queries. Currently:

- **Database**: Single `urls` table with columns `id`, `slug`, `original_url`, `click_count`, `created_at`
- **Store layer**: Wraps sqlc-generated `Queries` — exposes `InsertURL`, `GetBySlug`, `IncrementClicks`, `GetStats`
- **Service layer**: Business logic — `CreateShortURL`, `RedirectBySlug`, `GetStats`
- **Handler layer**: HTTP handlers — `HandleShorten` (POST /shorten), `HandleRedirect` (GET /{slug}), `HandleStats` (GET /{slug}/stats)
- **Routing**: Chi router with CORS allowing GET, POST, OPTIONS
- **Testing**: testify mocks at handler and service layers; SQLite in-memory integration tests at store layer

### Affected Areas

- `db/queries/urls.sql` — new sqlc query to list all slugs
- `internal/store/urls.sql.go` — auto-generated from sqlc (regenerated)
- `internal/store/querier.go` — auto-generated from sqlc (regenerated)
- `internal/store/store.go` — new `ListSlugs` method wrapping sqlc query
- `internal/service/interfaces.go` — add `ListSlugs` to Store interface + new response type
- `internal/service/service.go` — new `ListSlugs` method
- `internal/handler/handler.go` — new `HandleListSlugs` handler + response type
- `internal/handler/routes.go` — register `GET /slugs` route
- `internal/handler/handler_test.go` — new test cases for list slugs handler
- `internal/service/service_test.go` — new test cases for list slugs service
- `internal/store/store_test.go` — new test cases for list slugs store
- `openspec/specs/` — new spec domain `url-listing/spec.md`

### Routing Consideration

A **critical design constraint**: There is an existing `GET /{slug}` route for redirects (e.g., `GET /abc12`). Chi's router uses a radix tree where static path segments take priority over parameterized segments at the same depth. This means `GET /slugs` (literal) will correctly route to the list handler while `GET /abc12` (parameterized) routes to the redirect handler — the two can coexist safely.

Additionally, CORS middleware currently only allows `GET, POST, OPTIONS`. Since `GET` is already permitted, no CORS change is needed.

### Approaches

1. **Simple flat list — `GET /slugs`** (Recommended)
   - Returns all slugs with `slug`, `original_url`, `click_count`, `created_at`
   - No pagination, no filtering
   - Pros: Simplest to implement, follows existing patterns, zero query complexity
   - Cons: Unbounded response for very large datasets; no way to filter or search
   - Effort: Low — one new sqlc query, ~3 new methods, ~1 new route, ~1 new response type

2. **Paginated list — `GET /slugs?page=1&limit=20`**
   - Adds `LIMIT ? OFFSET ?` to the SQL query, accept page/limit query params
   - Pros: Scales to large datasets, standard API pattern
   - Cons: OFFSET-based pagination has drift on large tables; more params to validate; over-engineering for current scale
   - Effort: Medium

3. **Cursor-based pagination — `GET /slugs?cursor=abc12&limit=20`**
   - Uses `WHERE slug > ? ORDER BY slug LIMIT ?`
   - Pros: Stable pagination even with concurrent inserts; no drift
   - Cons: More complex API; clients must track opaque cursors; overkill for a URL shortener
   - Effort: Medium-High

### Recommendation

**Approach 1 — Simple flat list**. Rationale:

- The config describes this as a fresh project. A URL shortener at realistic scale will have thousands, not millions, of slugs. A flat list with no pagination is entirely appropriate.
- If pagination is needed later, it can be added as an OPTIONAL query parameter without breaking existing clients (default to no limit).
- Keeps the implementation aligned with existing patterns: one sqlc query, one store method, one service method, one handler.
- Minimum code change, maximum immediate value.

### Implementation Sketch

**SQL query** (`db/queries/urls.sql`):
```sql
-- name: ListSlugs :many
SELECT slug, original_url, click_count, created_at FROM urls
ORDER BY created_at DESC;
```

**Store interface addition** (`internal/service/interfaces.go`):
```go
type SlugEntry struct {
    Slug        string
    OriginalURL string
    ClickCount  int64
    CreatedAt   time.Time
}

type Store interface {
    InsertURL(ctx context.Context, slug, originalURL string) error
    GetBySlug(ctx context.Context, slug string) (string, error)
    IncrementClicks(ctx context.Context, slug string) error
    GetStats(ctx context.Context, slug string) (*URLStats, error)
    ListSlugs(ctx context.Context) ([]SlugEntry, error)  // NEW
}
```

**Service method**:
```go
func (s *Service) ListSlugs(ctx context.Context) ([]SlugEntry, error) {
    return s.store.ListSlugs(ctx)
}
```

**Handler** (`GET /slugs`):
```go
type SlugListEntry struct {
    Slug        string `json:"slug"`
    OriginalURL string `json:"original_url"`
    ClickCount  int64  `json:"click_count"`
    CreatedAt   string `json:"created_at"`
}

func (h *Handler) HandleListSlugs(w http.ResponseWriter, r *http.Request) {
    entries, err := h.svc.ListSlugs(r.Context())
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
        return
    }
    resp := make([]SlugListEntry, len(entries))
    for i, e := range entries {
        resp[i] = SlugListEntry{
            Slug:        e.Slug,
            OriginalURL: e.OriginalURL,
            ClickCount:  e.ClickCount,
            CreatedAt:   e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
        }
    }
    writeJSON(w, http.StatusOK, resp)
}
```

**Route**:
```go
r.Get("/slugs", h.HandleListSlugs)
```

### Risks

- **Route ordering not guaranteed by Chi**: Though Chi's radix tree favors static over parameterized, the `GET /{slug}` route is already registered. Should be tested explicitly in the handler test to confirm routing works. If Chi ever changes this behavior, the two routes would conflict.
- **No pagination**: A simple flat list becomes expensive if the user base grows large. Mitigation: add an optional `LIMIT` query parameter in a follow-up change.
- **Empty list response**: When no slugs exist, returning `[]` (empty array) is the expected correct behavior, not `null`.

### Ready for Proposal

Yes — the approach is clear, the routing concern is understood and safe (verified against Chi's radix tree behavior), and the implementation follows existing patterns exactly.