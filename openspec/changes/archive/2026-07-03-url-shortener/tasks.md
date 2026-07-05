# Tasks: URL Shortener Service

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 350–450 |
| 400-line budget risk | Medium |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | ask-on-risk |
| Chain strategy | size-exception |

Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Medium

## Phase 1: Foundation

- [x] 1.1 Create `go.mod` with module `github.com/url-shortener` and deps: chi, sqlc, testify, go-sqlite3
- [x] 1.2 Create `db/migrations/001_create_urls.sql` — `urls` table (id, slug UNIQUE, original_url, click_count DEFAULT 0, created_at) + slug index
- [x] 1.3 Create `db/queries/urls.sql` — sqlc queries: InsertURL, GetBySlug, IncrementClicks, GetStats
- [x] 1.4 Create `sqlc.yaml` — sqlc config pointing to db/; run `sqlc generate` to produce `internal/store/querier.go`
- [x] 1.5 Create `Makefile` with targets: build, test, run, sqlc-generate

## Phase 2: Store Layer (TDD)

- [x] 2.1 Write `internal/store/store_test.go` — tests for InsertURL, GetBySlug (found + not-found), IncrementClicks, GetStats using in-memory SQLite
- [x] 2.2 Create `internal/store/store.go` — implement `Store` struct wrapping sqlc `*Queries`; satisfy all test cases
- [x] 2.3 Verify all store tests pass with `go test ./internal/store/...`

## Phase 3: Service Layer (TDD)

- [x] 3.1 Create `internal/service/interfaces.go` — define `Store` interface (InsertURL, GetBySlug, IncrementClicks, GetStats) and `URLStats` struct
- [x] 3.2 Write `internal/service/service_test.go` — tests: valid URL → 201 + 5-char code; invalid URL → 400; empty body → 400; slug collision → retry; redirect found → 301; redirect not-found → 404; stats found → 200; stats not-found → 404; click increment on redirect; no increment on 404
- [x] 3.3 Create `internal/service/service.go` — implement `CreateShortURL`, `RedirectBySlug`, `GetStats` with validation, slug gen (crypto/rand), collision retry
- [x] 3.4 Verify all service tests pass with `go test ./internal/service/...`

## Phase 4: Handler Layer (TDD)

- [x] 4.1 Write `internal/handler/handler_test.go` — httptest integration: POST /shorten (201 + JSON, 400 invalid, 400 empty); GET /:slug (301 redirect, 404); GET /:slug/stats (200 + JSON, 404)
- [x] 4.2 Create `internal/handler/handler.go` — HTTP handlers: parse JSON, call service, write responses
- [x] 4.3 Create `internal/handler/routes.go` — Chi router: mount 3 routes, return `http.Handler`
- [x] 4.4 Verify all handler tests pass with `go test ./internal/handler/...`

## Phase 5: Integration & Wiring

- [x] 5.1 Create `cmd/url-shortener/main.go` — open SQLite (WAL mode), init store → service → handler, register routes, listen on :8080
- [x] 5.2 Run `go test ./...` — all tests green (cover store + service + handler = 34 tests)

## Phase 6: Cleanup

- [x] 6.1 Add `README.md` with build/run/test instructions and example curl commands
- [x] 6.2 Verify `make build && make test` succeeds cleanly