# Tasks: List All Slugs

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~150-170 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | single PR |
| Delivery strategy | single-pr |
| Chain strategy | not needed |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

## Phase 1: Foundation (SQL + Domain Types)

- [x] 1.1 Add `ListSlugs :many` query to `db/queries/urls.sql` — `SELECT slug, original_url, click_count, created_at FROM urls ORDER BY created_at DESC`
- [x] 1.2 Run `sqlc generate` to regenerate `internal/store/urls.sql.go` and `internal/store/querier.go`
- [x] 1.3 Add `SlugEntry` struct to `internal/service/interfaces.go` with json tags matching spec fields
- [x] 1.4 Add `ListSlugs(ctx context.Context) ([]SlugEntry, error)` to `Store` interface in `internal/service/interfaces.go`

## Phase 2: Core Implementation (Store + Service + Handler)

- [x] 2.1 Add `ListSlugs` method to `internal/store/store.go` — call `q.ListSlugs`, map each `ListSlugsRow` to `service.SlugEntry`, return slice
- [x] 2.2 Add `ListSlugs` method to `internal/service/service.go` — thin pass-through delegating to `store.ListSlugs`
- [x] 2.3 Add `SlugListEntry` type to `internal/handler/handler.go` with `CreatedAt` as RFC3339 string (match `StatsResponse` pattern)
- [x] 2.4 Add `HandleListSlugs` to `internal/handler/handler.go` — call `svc.ListSlugs`, map to `[]SlugListEntry`, use `make([]T, 0)` for empty response, write 200 JSON

## Phase 3: Integration (Route Registration)

- [x] 3.1 Add `r.Get("/slugs", h.HandleListSlugs)` in `internal/handler/routes.go` BEFORE the `/{slug}` route (static before parameterized)

## Phase 4: Testing

- [x] 4.1 Add `ListSlugs` method to both `mockStore` implementations in `internal/handler/handler_test.go` and `internal/service/service_test.go`
- [x] 4.2 Add handler tests in `internal/handler/handler_test.go` — 200 with JSON array, empty returns `[]` not `null`, `/slugs` returns listing not redirect (coexistence with `/{slug}`)
- [x] 4.3 Add service tests in `internal/service/service_test.go` — success returns slice, store error propagates
- [x] 4.4 Add integration tests in `internal/store/store_test.go` — multiple URLs returned in `created_at DESC` order, empty DB returns empty slice
