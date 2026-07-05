# Design: Paginate List Slugs

## Technical Approach

Replace the unbounded `GET /slugs` with offset-based pagination. Two new SQL queries (`CountSlugs`, `ListSlugsPaginated`) feed a new service method that validates params, wraps errors with `fmt.Errorf`, and returns a `{ data, pagination }` envelope. Handler parses `page`/`limit` query params. All three test layers updated.

## Architecture Decisions

### Decision: Offset-based pagination

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Offset (`LIMIT ? OFFSET ?`) | Simple, jump-to-page; scans skipped rows | **Chosen** |
| Cursor (`WHERE created_at < ?`) | Stable under writes; no arbitrary page jump | Rejected |
| Simple LIMIT only | Minimal change; no navigation metadata | Rejected |

**Rationale**: Fresh project at low scale. SQLite handles OFFSET fine at tens of thousands of rows.

### Decision: Two queries per request

| Option | Tradeoff | Decision |
|--------|----------|----------|
| COUNT + data queries | Accurate total; extra round-trip | **Chosen** |
| Single query, estimated total | Faster; stale metadata | Rejected |

**Rationale**: SQLite `COUNT(*)` on primary key is sub-millisecond. Accuracy matters for pagination metadata.

### Decision: Replace handler in-place

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Replace in-place | Breaking change; clean API | **Chosen** |
| Conditional envelope (flat when no params) | Backward-compatible; two response shapes | Rejected |

**Rationale**: Fresh project, no consumers. Clean break over dual-format complexity.

### Decision: Error wrapping

**Choice**: New method wraps both store errors with `fmt.Errorf("failed to ...: %w", err)` — matching `CreateShortURL` (line 81) and `GetStats` (line 112) patterns. Old `ListSlugs` removed.

## Data Flow

```
Client → Handler.Parse(page, limit) → Service.ListSlugsPaginated(ctx, page, limit)
                                            ├→ Store.CountSlugs → SELECT COUNT(*)
                                            └→ Store.ListSlugsPaginated → SELECT ... LIMIT ? OFFSET ?
                                            → compute totalPages → Handler writes { data, pagination }
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `db/queries/urls.sql` | Modify | Add `ListSlugsPaginated` and `CountSlugs` queries |
| `internal/store/urls.sql.go` | Regenerate | sqlc auto-generates new query functions |
| `internal/store/querier.go` | Regenerate | sqlc auto-adds to Querier interface |
| `internal/store/store.go` | Modify | Add `ListSlugsPaginated` and `CountSlugs` wrappers; remove `ListSlugs` |
| `internal/service/interfaces.go` | Modify | Add `PaginationMeta`, `PaginatedSlugs` types; update `Store` interface |
| `internal/service/service.go` | Modify | Replace `ListSlugs` with `ListSlugsPaginated` + param validation + error wrapping |
| `internal/handler/handler.go` | Modify | Parse query params, call new service method, return envelope |
| `internal/handler/handler_test.go` | Modify | Paginated test cases; update mock for new interface |
| `internal/service/service_test.go` | Modify | Paginated test cases; update mock for new interface |
| `internal/store/store_test.go` | Modify | Tests for `CountSlugs` and `ListSlugsPaginated` |

## Interfaces / Contracts

### New Types (service/interfaces.go)

```go
type PaginationMeta struct {
    Page       int   `json:"page"`
    Limit      int   `json:"limit"`
    Total      int64 `json:"total"`
    TotalPages int   `json:"total_pages"`
}

type PaginatedSlugs struct {
    Data       []SlugEntry    `json:"data"`
    Pagination PaginationMeta `json:"pagination"`
}
```

### Updated Store Interface

```go
ListSlugsPaginated(ctx context.Context, limit, offset int) ([]SlugEntry, error)
CountSlugs(ctx context.Context) (int64, error)
// Replaces: ListSlugs(ctx context.Context) ([]SlugEntry, error)
```

### Response Envelope

```json
{
  "data": [{ "slug": "...", "original_url": "...", "click_count": 0, "created_at": "..." }],
  "pagination": { "page": 1, "limit": 50, "total": 100, "total_pages": 2 }
}
```

### SQL Queries

```sql
-- name: ListSlugsPaginated :many
SELECT slug, original_url, click_count, created_at FROM urls
ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountSlugs :one
SELECT COUNT(*) FROM urls;
```

### Query Param Defaults

| Param | Default | Validation |
|-------|---------|------------|
| `page` | 1 | `< 1` clamped to 1 |
| `limit` | 50 | `< 1` or `> 200` clamped to 50 |

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Store | `CountSlugs` count; `ListSlugsPaginated` correct page slice | Real SQLite in-memory DB (`setupTestDB`) |
| Service | Param clamping; `fmt.Errorf` wrapping; pagination math | Testify mock store |
| Handler | Query param defaults; `?page=2&limit=2`; envelope shape; empty result; store error → 500 | Testify mock + httptest |

## Migration / Rollout

No migration required. Breaking API change on a fresh project with no consumers. Old `ListSlugs` removed entirely — no parallel endpoints.

## Open Questions

None — all decisions resolved during exploration.
