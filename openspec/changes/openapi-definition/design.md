# Design: Add OpenAPI Definition

## Technical Approach

Embed a static `api/openapi.yaml` file using Go's `//go:embed` directive and serve it at `GET /openapi.yaml` via the existing Chi router. The embed and handler registration live in `internal/handler/routes.go` — no new packages, no new dependencies. Route ordering ensures `/openapi.yaml` is matched before the `/{slug}` wildcard.

## Architecture Decisions

### Decision: Embed location in `routes.go`

| Option | Tradeoff | Decision |
|--------|----------|----------|
| New `api` package for embed | Clean separation but adds a package for 3 lines of code | Rejected |
| `routes.go` embed + handler | Co-located with route registration, minimal diff, follows existing pattern | **Selected** |
| `main.go` embed + pass bytes | Pushes routing concern into main, breaks encapsulation | Rejected |

**Rationale**: `routes.go` already owns route registration and is the natural home for a new route + its backing embed. The file is 31 lines — adding ~15 lines keeps it readable.

### Decision: `application/x-yaml` content type

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `application/x-yaml` | Non-standard but widely recognized by OpenAPI tooling (Swagger UI, Redoc, VS Code) | **Selected** |
| `text/yaml` | RFC-standard but some OpenAPI tools don't auto-detect | Rejected |
| `application/yaml` | RFC 9512 standard, but newer — tooling support varies | Rejected |

**Rationale**: `application/x-yaml` is the de facto standard for OpenAPI YAML files. All major tools (Swagger UI, Redoc, `curl`, VS Code OpenAPI extensions) handle it correctly.

### Decision: Route ordering — static before wildcard

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Register `/openapi.yaml` before `/{slug}` | Chi matches first-registered route; zero risk of conflict | **Selected** |
| Use Chi's `Route` group with explicit ordering | Over-engineered for one static route | Rejected |

**Rationale**: Chi evaluates routes in registration order. A literal path like `/openapi.yaml` registered before `/{slug}` will always match correctly. This is the documented Chi pattern for static routes alongside wildcards.

## Data Flow

```
Client                    Chi Router                Handler
  │                          │                         │
  ├─ GET /openapi.yaml ─────►│                         │
  │                          ├─ match literal route    │
  │                          ├─ serveOpenAPI() ────────┤
  │                          │   read embed.FS         │
  │  ◄── 200 + YAML body ───┤   write Content-Type    │
  │                          │   write bytes           │
  │                          │                         │
  ├─ GET /abc12 ────────────►│                         │
  │                          ├─ match /{slug}          │
  │                          ├─ HandleRedirect() ──────┤
  │  ◄── 301 Location ──────┤                         │
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `api/openapi.yaml` | Create | OpenAPI 3.1.0 spec covering all 4 endpoints, schemas, error responses, CORS headers |
| `internal/handler/routes.go` | Modify | Add `embed` import, `//go:embed` directive, `serveOpenAPI` handler, register `GET /openapi.yaml` before wildcard |

## Interfaces / Contracts

No new Go interfaces. The changes are additive:

```go
// In routes.go — new embed and handler

import "embed"

//go:embed api/openapi.yaml
var openapiSpec []byte

func serveOpenAPI(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/x-yaml")
    w.WriteHeader(http.StatusOK)
    w.Write(openapiSpec) //nolint:errcheck
}
```

Route registration order (modified):
```go
r.Get("/openapi.yaml", serveOpenAPI)  // NEW — before wildcard
r.Get("/slugs", h.HandleListSlugs)
r.Get("/{slug}", h.HandleRedirect)
r.Get("/{slug}/stats", h.HandleStats)
```

**Important**: The `//go:embed` directive in `routes.go` resolves relative to the **package directory** (`internal/handler/`), not the repo root. The YAML file must be either:
- At `internal/handler/openapi.yaml` (co-located), OR
- The embed must live in a file at or above the `api/` directory level

Since `api/` is at the repo root and `routes.go` is under `internal/handler/`, we need to place the embed directive in a file at the repo root or use a root-level package. The simplest approach: **put the embed in a small `api/embed.go` file** that exports the bytes, and import it from `routes.go`.

```go
// api/embed.go
package api

import _ "embed"

//go:embed openapi.yaml
var OpenAPISpec []byte
```

```go
// In routes.go
import "github.com/url-shortener/api"

func serveOpenAPI(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/x-yaml")
    w.WriteHeader(http.StatusOK)
    w.Write(api.OpenAPISpec) //nolint:errcheck
}
```

This keeps the embed co-located with the YAML file (correct `//go:embed` resolution) and the route handler in `routes.go` (where it belongs).

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `serveOpenAPI` returns correct content-type and body | `httptest.NewRecorder` + direct call |
| Integration | `GET /openapi.yaml` returns 200 + valid YAML | `httptest.NewServer` with full router, verify status + content-type header |
| Spec validity | YAML is valid OpenAPI 3.1 | Manual review or future CI with `swagger-cli validate` (out of scope now) |

## Migration / Rollout

No migration required. Purely additive — two new files, one modified file. No database or config changes.

## Open Questions

- [ ] None — all technical decisions are resolved.
