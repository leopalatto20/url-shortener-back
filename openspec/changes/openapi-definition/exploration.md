# Exploration: openapi-definition

## Current State

The URL shortener has 4 endpoints implemented in Go using the Chi router, with full specs defined under `openspec/specs/`. There is no OpenAPI/Swagger documentation anywhere in the project — no swagger files, no OpenAPI YAML/JSON, and no code annotations.

### Existing Endpoints

| Method | Path | Handler | Status |
|--------|------|---------|--------|
| POST | `/shorten` | `HandleShorten` | 201 / 400 / 500 |
| GET | `/{slug}` | `HandleRedirect` | 301 / 404 / 500 |
| GET | `/{slug}/stats` | `HandleStats` | 200 / 404 / 500 |
| GET | `/slugs` | `HandleListSlugs` | 200 / 400 / 500 |

### Request/Response Shapes

**POST /shorten**
- Request: `{"url": "https://..."}`
- 201: `{"short_code": "abc12", "short_url": "http://localhost:8080/abc12"}`
- 400: `{"error": "..."}` — on empty body, invalid JSON, missing URL, or non-HTTP URL

**GET /{slug}**
- 301: Redirect with `Location` header, no body
- 404: `{"error": "short URL not found"}`

**GET /{slug}/stats**
- 200: `{"original_url": "...", "click_count": 5, "created_at": "2024-01-01T00:00:00Z"}`
- 404: `{"error": "short URL not found"}`

**GET /slugs?page=1&limit=50**
- Query params: `page` (default 1, min 1), `limit` (default 50, max 200, min 1)
- 200: `{"data": [...], "pagination": {"page": 1, "limit": 50, "total": 120, "total_pages": 3}}`
- 400: `{"error": "invalid page parameter"}` or `{"error": "invalid limit parameter"}`
- Response `created_at` format: RFC3339 (Go `time.RFC3339`)

### Error Response Format (all endpoints)

```json
{"error": "descriptive message"}
```

### CORS Configuration

```go
AllowedOrigins: []string{"http://localhost:5173"},
AllowedMethods: []string{"GET", "POST", "OPTIONS"},
AllowedHeaders: []string{"Accept", "Content-Type"},
ExposedHeaders: []string{"Location"},
```

### Server Info

- Base URL: `http://localhost:8080` (configurable via `BASE_URL` env)
- Framework: Chi v5 (`github.com/go-chi/chi/v5`)
- CORS: `github.com/go-chi/cors`

### UUID / Schema

Database table `urls`:
- `slug` TEXT NOT NULL UNIQUE (5-char alphanumeric, PK via unique index)
- `original_url` TEXT NOT NULL
- `click_count` INTEGER NOT NULL DEFAULT 0
- `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP

## Affected Areas

No existing code needs modification — this is purely additive. The OpenAPI spec lives independently.

## Approaches

### 1. Static OpenAPI YAML file (recommended)

A standalone `openapi.yaml` file in the project root or `api/` directory.

- **Pros**: Zero code changes, no dependencies, version-controlled, readable by any OpenAPI tool, can be served statically
- **Cons**: Must be manually kept in sync with code (no auto-generation)
- **Effort**: Low

### 2. Go code annotations (swaggo/swag)

Annotate handler functions with comments, run `swag init` to generate.

- **Pros**: Auto-generated from code, stays close to handlers
- **Cons**: Adds dependency (`swaggo/swag`, `swaggo/http-swagger`), requires comment maintenance anyway, Go annotation syntax is verbose, coupled to one specific framework
- **Effort**: Medium

### 3. Serve spec via endpoint (GET /openapi.yaml at runtime)

Embed the YAML file and serve it at `/openapi.yaml`.

- **Pros**: Self-documenting API, one URL to point Swagger UI at
- **Cons**: Adds a runtime handler for a build-time concern, tight coupling
- **Effort**: Low (trivial embedding)

### 4. Full Swagger UI setup

Serve a `/docs` or `/swagger` page with Swagger UI loading the OpenAPI spec.

- **Pros**: Interactive API exploration, great DX for frontend devs
- **Cons**: Requires embedding Swagger UI assets or pulling from CDN, adds complexity
- **Effort**: Medium

### 5. Hybrid: static file + serve endpoint

Create the static YAML file AND serve it via a small endpoint.

- **Pros**: Best of both — versioned file for CI/tooling and runtime discoverability
- **Cons**: Two things to maintain (but they're the same content)
- **Effort**: Low

## Recommendation

**Approach 5 (Hybrid: static YAML + serve endpoint)** — create the OpenAPI 3.1 YAML file at `api/openapi.yaml`, embed it using Go's `//go:embed` directive, and mount a `GET /openapi.yaml` handler. This gives frontend a single URL to fetch the spec, keeps the source in version control, and requires only ~10 extra lines of Go code.

OpenAPI 3.1 over 3.0: 3.1 is JSON Schema compatible, supports `webhooks`, and is the current OpenAPI standard.

## Format Choices

- **Format**: YAML (more human-readable than JSON for API definitions)
- **Version**: OpenAPI 3.1.0 (JSON Schema 2020-12 compatibility, modern tooling support)
- **Location**: `api/openapi.yaml` (separate from source code, conventional location)
- **Serve endpoint**: `GET /openapi.yaml` with `Content-Type: application/x-yaml`

## Risks

- **Manual sync risk**: The spec must be updated whenever endpoints change. Mitigation: place the OpenAPI file as the single source of truth for API contracts alongside the existing specs, and document in the spec that it must be updated in parallel with endpoint changes.
- **Route conflict**: `GET /openapi.yaml` could theoretically conflict with `GET /{slug}` if someone creates a slug named `openapi.yaml`. This is extremely unlikely and would only affect that one slug. Mitigation: register `/openapi.yaml` BEFORE the wildcard `/{slug}` route in Chi, or use Chi's route ordering (specific routes before wildcards).

## Ready for Proposal

Yes — all endpoints, request/response shapes, query parameters, error formats, and CORS configuration are fully documented. The orchestrator should proceed with `sdd-propose`.
