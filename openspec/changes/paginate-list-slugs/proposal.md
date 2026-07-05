# Proposal: Paginate List Slugs

## Intent

Fix two review findings on `GET /slugs`: (1) unbounded SQL query with no `LIMIT` risks OOM at scale, (2) `Service.ListSlugs` passes raw store errors without wrapping. Add offset-based pagination with a response envelope.

## Scope

### In Scope
- Add `page` and `limit` query params to `GET /slugs` (defaults: 1, 50; max limit: 200)
- Change response from flat JSON array to `{ data, pagination }` envelope
- Add `ListSlugsPaginated` and `CountSlugs` SQL queries (sqlc)
- Add paginated service method with `fmt.Errorf` error wrapping
- Update handler to parse query params and return envelope
- Update all tests (handler, service, store)
- Update `openspec/specs/url-listing/spec.md` to reflect new contract

### Out of Scope
- Cursor-based pagination (deferred — offset is sufficient at current scale)
- Caching the COUNT query (SQLite COUNT is sub-millisecond at this scale)
- Archiving old `list-slugs` change (separate lifecycle step)

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `url-listing`: Response format changes from flat array to `{ data, pagination }` envelope. Adds `page` and `limit` query params. Adds error wrapping in service layer.

## Approach

Offset-based pagination with two SQL queries per request:

1. `CountSlugs` — `SELECT COUNT(*) FROM urls` for total
2. `ListSlugsPaginated` — `SELECT ... FROM urls ORDER BY created_at DESC LIMIT ? OFFSET ?`

Service validates params (clamp page ≥ 1, limit 1–200), computes offset, wraps both errors with `fmt.Errorf`. Handler parses `?page=` and `?limit=` from query string, returns envelope JSON.

Response envelope:
```json
{
  "data": [{ "slug": "...", "original_url": "...", "click_count": 0, "created_at": "..." }],
  "pagination": { "page": 1, "limit": 50, "total": 100, "total_pages": 2 }
}
```

Breaking change is acceptable — project is fresh with no real consumers.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `db/queries/urls.sql` | Modified | Add `ListSlugsPaginated` and `CountSlugs` queries |
| `internal/store/urls.sql.go` | Regenerated | sqlc output |
| `internal/store/store.go` | Modified | New store methods |
| `internal/service/interfaces.go` | Modified | Add paginated types + methods to interface |
| `internal/service/service.go` | Modified | New method with validation + error wrapping |
| `internal/handler/handler.go` | Modified | Query param parsing, envelope response |
| `internal/handler/handler_test.go` | Modified | Paginated test cases |
| `internal/service/service_test.go` | Modified | Paginated test cases |
| `internal/store/store_test.go` | Modified | New query tests |
| `openspec/specs/url-listing/spec.md` | Modified | New response format + pagination requirement |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| OFFSET drift on concurrent inserts | Low | Append-mostly workload; acceptable |
| OFFSET perf at high page numbers | Low | Default limit 50 caps scan; SQLite handles tens of thousands fine |
| Two queries per request adds latency | Low | COUNT on primary key is sub-ms |

## Rollback Plan

Revert handler to return flat `[]SlugEntry` array. Remove `page`/`limit` params. Restore original `ListSlugs` service method without error wrapping. Re-run `sqlc generate` to remove new queries. Revert spec to flat array contract.

## Dependencies

- None (all changes are internal to this project)

## Success Criteria

- [x] `GET /slugs` returns envelope with `data` and `pagination` fields
- [x] `GET /slugs?page=2&limit=10` returns correct page slice
- [x] Default request (no params) returns page 1, limit 50
- [x] `limit` values > 200 are clamped to 200; < 1 clamped to 50
- [x] Service errors wrapped with `fmt.Errorf` context
- [x] All existing tests pass with updated assertions
- [x] No unbounded `SELECT` without `LIMIT` in the codebase
