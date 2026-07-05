# Tasks: Paginate List Slugs

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~280 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | single PR |
| Delivery strategy | single-pr |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: stacked-to-main
400-line budget risk: Low

## Phase 1: SQL Layer (Foundation)

- [x] 1.1 Replace `ListSlugs` query in `db/queries/urls.sql` with `ListSlugsPaginated :many` using `LIMIT ? OFFSET ?` and add `CountSlugs :one` (`SELECT COUNT(*) FROM urls`)
- [x] 1.2 Run `sqlc generate` to regenerate `internal/store/urls.sql.go` and `internal/store/querier.go`

## Phase 2: Types & Interfaces

- [x] 2.1 Add `PaginationMeta` and `PaginatedSlugs` structs to `internal/service/interfaces.go`
- [x] 2.2 Replace `ListSlugs` with `ListSlugsPaginated(ctx, limit, offset int)` and add `CountSlugs(ctx)` in `Store` interface (`internal/service/interfaces.go`)

## Phase 3: Store Layer

- [x] 3.1 Replace `ListSlugs` method with `ListSlugsPaginated` and add `CountSlugs` in `internal/store/store.go`; remove old `ListSlugs`

## Phase 4: Service Layer

- [x] 4.1 Replace `ListSlugs` with `ListSlugsPaginated` in `internal/service/service.go`: validate params (clamp page≥1, limit 1–200), compute offset, call both store methods, wrap errors with `fmt.Errorf`, return `PaginatedSlugs`

## Phase 5: Handler Layer

- [x] 5.1 Add `SlugsResponse` envelope struct and `PaginationResponse` to `internal/handler/handler.go`
- [x] 5.2 Rewrite `HandleListSlugs` to parse `page`/`limit` query params with `strconv.Atoi`, call `ListSlugsPaginated`, return `{ data, pagination }` envelope

## Phase 6: Tests

- [x] 6.1 Add `ListSlugsPaginated` and `CountSlugs` mock methods to `mockStore` in `internal/handler/handler_test.go` and `internal/service/service_test.go`
- [x] 6.2 Update handler tests: default params → page 1/limit 50, explicit `?page=2&limit=2`, empty result, store error → 500, envelope shape validation
- [x] 6.3 Update service tests: param clamping (page<1, limit<1, limit>200), error wrapping with `errors.As`/`errors.Is`, pagination math (total_pages), two-query call verification
- [x] 6.4 Replace store `ListSlugs` tests with `CountSlugs` and `ListSlugsPaginated` tests: correct count, correct page slice, empty DB, offset correctness
- [x] 6.5 Run `go test ./...` — all tests pass

## Phase 7: Spec Sync

- [x] 7.1 Verify `openspec/specs/url-listing/spec.md` is updated with paginated response format (delta already prepared in specs/)
