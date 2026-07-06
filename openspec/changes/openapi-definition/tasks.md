# Tasks: Add OpenAPI Definition

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~300 (YAML ~200, Go ~40, tests ~60) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | single PR |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

## Phase 1: Foundation — OpenAPI Spec

- [x] 1.1 Create `api/openapi.yaml` with OpenAPI 3.1.0 header: `openapi`, `info` (title: "URL Shortener API", version: "1.0.0"), `servers` (localhost:8080)
- [x] 1.2 Add `components/schemas` section: `ShortenRequest` (url: string, required), `ShortenResponse` (short_code, short_url), `StatsResponse` (original_url, click_count: integer, created_at: date-time), `SlugEntry` (slug, original_url, click_count: integer, created_at: date-time), `PaginationMeta` (page, limit, total, total_pages — all integer), `SlugsResponse` (data: array of SlugEntry, pagination: PaginationMeta), `ErrorResponse` (error: string)
- [x] 1.3 Add `POST /shorten` path: request body requiring `url` (string), responses for 201 (ShortenResponse), 400 (ErrorResponse), 500 (ErrorResponse)
- [x] 1.4 Add `GET /{slug}` path: required `slug` path param, responses for 301 (with `Location` header), 404 (ErrorResponse), 500 (ErrorResponse)
- [x] 1.5 Add `GET /{slug}/stats` path: required `slug` path param, responses for 200 (StatsResponse), 404 (ErrorResponse), 500 (ErrorResponse)
- [x] 1.6 Add `GET /slugs` path: query params `page` (integer, default 1, minimum 1) and `limit` (integer, default 50, minimum 1, maximum 200), responses for 200 (SlugsResponse), 400 (ErrorResponse), 500 (ErrorResponse)

## Phase 2: Wiring — Embed and Serve

- [x] 2.1 Create `api/embed.go`: package `api`, import `_ "embed"`, `//go:embed openapi.yaml` directive, export `var OpenAPISpec []byte`
- [x] 2.2 Modify `internal/handler/routes.go`: add import `"github.com/url-shortener/api"` to use the exported `api.OpenAPISpec` bytes
- [x] 2.3 Add `serveOpenAPI` handler function in `routes.go`: sets `Content-Type: application/x-yaml`, writes `api.OpenAPISpec` bytes with status 200
- [x] 2.4 Register `r.Get("/openapi.yaml", serveOpenAPI)` **before** the `/{slug}` and `/{slug}/stats` wildcard routes — this is critical to prevent route conflicts

## Phase 3: Testing

- [x] 3.1 Add `TestHandler_OpenAPI_Returns200` in `handler_test.go`: `GET /openapi.yaml` via full router, assert status 200 and `Content-Type: application/x-yaml`
- [x] 3.2 Add `TestHandler_OpenAPI_BodyIsValidYAML`: parse response body as YAML, assert `openapi` field equals `"3.1.0"`, assert `paths` contains `/shorten`, `/{slug}`, `/{slug}/stats`, `/slugs`
- [x] 3.3 Add `TestHandler_OpenAPI_RouteNotConflicting`: verify `GET /openapi.yaml` returns 200 (not 301 redirect or 404) — confirms literal route wins over wildcard
- [x] 3.4 Run full test suite: `go test ./...` — all existing + new tests pass

## Phase 4: Verification

- [x] 4.1 Manual curl check: `curl -i localhost:8080/openapi.yaml` returns 200 with YAML content-type
- [x] 4.2 Validate YAML structure: all 4 endpoints present, shared ErrorResponse `$ref` used for error codes, `created_at` is `format: date-time`, `click_count` is `integer`
