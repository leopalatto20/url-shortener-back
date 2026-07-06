# URL Shortener Service

A minimalist URL shortener built with Go, Chi, sqlc, and SQLite.

## Quick Start

```bash
# Build
make build

# Run tests
make test

# Start the server
make run
```

The server starts on `:8080` by default. Configure via environment variables:

| Variable     | Default                  | Description               |
|-------------|--------------------------|---------------------------|
| `LISTEN_ADDR` | `:8080`                | Server listen address     |
| `BASE_URL`    | `http://localhost:8080` | Base URL for short links  |
| `DB_PATH`     | `url-shortener.db`     | SQLite database file path |

## API

### Create a short URL

```bash
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/very/long/url"}'
```

Response: `201 Created`
```json
{
  "short_code": "aB3xY",
  "short_url": "http://localhost:8080/aB3xY"
}
```

### Redirect to original URL

```bash
curl -v http://localhost:8080/aB3xY
```

Response: `301 Moved Permanently` with `Location` header pointing to the original URL.

### Get URL stats

```bash
curl http://localhost:8080/aB3xY/stats
```

Response: `200 OK`
```json
{
  "original_url": "https://example.com/very/long/url",
  "click_count": 3,
  "created_at": "2026-07-03T16:30:00Z"
}
```

### List all slugs

```bash
curl http://localhost:8080/slugs
```

Query parameters:
- `page` — page number (default: 1)
- `limit` — results per page (default: 50, min: 1, max: 200)

```bash
curl http://localhost:8080/slugs?page=2&limit=10
```

Response: `200 OK`
```json
{
  "data": [
    {
      "slug": "aB3xY",
      "original_url": "https://example.com/very/long/url",
      "click_count": 3,
      "created_at": "2026-07-03T16:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 1,
    "total_pages": 1
  }
}
```

Returns an empty `data` array when no short URLs exist. Entries are ordered by creation date (newest first).

### OpenAPI specification

The full OpenAPI 3.1 spec is served at runtime:

```bash
curl http://localhost:8080/openapi.yaml
```

Response: `200 OK` with `Content-Type: application/x-yaml`.

### Error responses

Invalid URLs return `400 Bad Request`:
```json
{
  "error": "invalid URL: must be a valid HTTP or HTTPS URL"
}
```

Unknown slugs return `404 Not Found`:
```json
{
  "error": "short URL not found"
}
```

Invalid pagination parameters return `400 Bad Request`:
```json
{
  "error": "invalid page parameter"
}
```

## Project Structure

```
├── cmd/url-shortener/main.go    # Entry point, wiring, graceful shutdown
├── api/
│   ├── openapi.yaml             # OpenAPI 3.1 specification
│   └── embed.go                 # Embeds spec for runtime serving
├── internal/
│   ├── handler/                 # HTTP handlers + Chi router
│   │   ├── handler.go
│   │   └── routes.go
│   ├── service/                 # Business logic
│   │   ├── service.go
│   │   └── interfaces.go
│   └── store/                   # SQLite data access (sqlc-generated)
│       ├── store.go
│       ├── db.go
│       ├── models.go
│       ├── querier.go
│       └── urls.sql.go
├── db/
│   ├── migrations/              # SQLite schema
│   └── queries/                 # sqlc query definitions
├── sqlc.yaml
├── Makefile
└── go.mod
```

## Architecture

Three-layer architecture:

- **Handler** — HTTP concerns (JSON parsing, response writing, route matching via Chi)
- **Service** — Business logic (URL validation, slug generation with `crypto/rand`, collision retry, click tracking)
- **Store** — Data access (sqlc-generated type-safe SQLite queries; `:memory:` for tests)

## Design Decisions

| Decision | Choice |
|----------|--------|
| Router | Chi v5 — middleware ecosystem, clean param extraction |
| Database | SQLite via mattn/go-sqlite3 (CGO) |
| Query layer | sqlc — type-safe generated Go from SQL |
| Slug generation | 5-char alphanumeric via `crypto/rand` |
| Testing | testify — readable assertions and mocks |
| Deployment | Single binary via `go build` |