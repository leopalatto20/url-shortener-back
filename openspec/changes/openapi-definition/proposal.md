# Proposal: Add OpenAPI Definition

## Intent

The URL shortener has zero API documentation. Frontend devs, consumers, and tooling have no machine-readable contract for the 4 existing endpoints. This adds an OpenAPI 3.1 spec and serves it at `GET /openapi.yaml` so the API is self-documenting at runtime.

## Scope

### In Scope
- Create `api/openapi.yaml` covering all 4 endpoints (`POST /shorten`, `GET /{slug}`, `GET /{slug}/stats`, `GET /slugs`)
- Embed the YAML via `//go:embed` and mount `GET /openapi.yaml` in the Chi router
- Document request/response schemas, status codes, query params, CORS headers, and error format

### Out of Scope
- Swagger UI / interactive docs page (future enhancement)
- Auto-generation from code annotations (`swaggo/swag`) — too much coupling for this stage
- CI validation of spec-vs-code drift

## Capabilities

### New Capabilities
- `api-documentation`: OpenAPI 3.1 spec for all endpoints, served at runtime via `GET /openapi.yaml`

### Modified Capabilities
None — no existing spec requirements change.

## Approach

1. Create `api/openapi.yaml` — OpenAPI 3.1.0 YAML documenting all endpoints, schemas, and error responses from the exploration
2. Add `//go:embed api/openapi.yaml` in a new `api` package or directly in the router setup
3. Register `GET /openapi.yaml` handler that serves the embedded bytes with `Content-Type: application/x-yaml`
4. Register the `/openapi.yaml` route **before** the `/{slug}` wildcard to prevent route conflicts

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `api/openapi.yaml` | New | OpenAPI 3.1 spec file |
| `main.go` or router setup | Modified | Add ~10 lines: embed + handler + route registration |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Spec drifts from code | Medium | Document in spec header that it must be updated with endpoint changes; future CI check |
| Route conflict with `/{slug}` | Low | Register `/openapi.yaml` before wildcard route in Chi |

## Rollback Plan

1. Delete `api/openapi.yaml`
2. Remove the embed directive and `GET /openapi.yaml` route registration (~10 lines)
3. No database or config changes to revert

## Dependencies

None — no new Go modules or external tools required.

## Success Criteria

- [ ] `GET /openapi.yaml` returns valid OpenAPI 3.1 YAML with `Content-Type: application/x-yaml`
- [ ] All 4 endpoints documented with correct request/response schemas and status codes
- [ ] Existing tests still pass unchanged
