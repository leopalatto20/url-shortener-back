# Proposal: URL Shortener Service

## Intent

Build a minimal URL shortener as the first SDD project — a real, working service to validate the SDD workflow end-to-end. The service accepts long URLs, returns short codes, redirects visitors, and tracks click stats.

## Scope

### In Scope
- `POST /shorten` — accept a URL, return auto-generated short code
- `GET /:slug` — 301 redirect to original URL
- `GET /:slug/stats` — return click count and creation timestamp
- SQLite storage via sqlc (type-safe queries)
- Standard Go testing with testify (TDD-first)
- Layered project structure (handler → service → store)

### Out of Scope
- Authentication / API keys
- Custom domains or vanity slugs
- Link expiration / TTL
- Rate limiting
- QR code generation
- Analytics beyond click count
- Bulk operations
- Web UI

## Capabilities

> Contract between proposal and specs phases.

### New Capabilities
- `url-creation`: Accept a long URL, validate it, generate a unique short code, persist the mapping, and return the short URL
- `url-redirect`: Resolve a short code to its original URL and issue a 301 redirect; return 404 for unknown slugs
- `url-stats`: Return click count and creation timestamp for a given short code

### Modified Capabilities
None — greenfield project.

## Approach

**Stack**: Go + Chi (router) + sqlc (type-safe SQL) + SQLite

**Architecture**: Three-layer structure per capability:
- `internal/handler/` — HTTP request/response, routing via Chi
- `internal/service/` — business logic, validation
- `internal/store/` — sqlc-generated queries against SQLite

**Slug generation**: Random 5-character alphanumeric code using `crypto/rand`. Collision check on insert; retry on conflict.

**Testing strategy**: TDD — write failing tests first, implement to pass. Use `go test` + testify assertions. Test at service layer (unit) and handler layer (integration with httptest).

**Project layout**:
```
cmd/server/main.go
internal/handler/
internal/service/
internal/store/
sqlc/sqlc.yaml
sqlc/queries/
```

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/server/main.go` | New | Entry point, router setup, server start |
| `internal/handler/` | New | HTTP handlers for all 3 endpoints |
| `internal/service/` | New | Business logic: validation, slug gen, stats |
| `internal/store/` | New | sqlc queries + SQLite schema |
| `sqlc/` | New | sqlc config, SQL migrations, query definitions |
| `go.mod` | New | Module definition and dependencies |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| SQLite write contention under concurrent requests | Low | WAL mode; acceptable for MVP |
| Slug collision on high volume | Very Low | Retry with new random code (5 chars = ~60M combos) |
| Scope creep into auth/analytics | Medium | Strict non-goals list; reject during review |

## Rollback Plan

Greenfield — delete the project directory. No existing data or services to restore.

## Dependencies

- Go 1.21+
- `github.com/go-chi/chi/v5`
- `github.com/sqlc-dev/sqlc`
- `github.com/stretchr/testify`
- `github.com/mattn/go-sqlite3`

## Success Criteria

- [ ] `POST /shorten` with a URL returns a short code
- [ ] `GET /:slug` returns 301 redirect to original URL
- [ ] `GET /:slug/stats` returns click count and created_at
- [ ] Unknown slug returns 404
- [ ] All tests pass (`go test ./...`)
- [ ] Service is demonstrable via `curl`
