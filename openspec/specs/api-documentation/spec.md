# API Documentation Specification

## Purpose

Machine-readable OpenAPI 3.1.0 spec covering all URL shortener endpoints, served at runtime for API discoverability.

## Requirements

### Requirement: OpenAPI Format

The spec MUST be valid OpenAPI 3.1.0 YAML with `info`, `servers`, `paths`, and `components` sections.

#### Scenario: Valid OpenAPI document

- GIVEN the spec file at `api/openapi.yaml`
- WHEN parsed as YAML
- THEN `openapi` SHALL equal `3.1.0`
- AND `info` SHALL contain `title` and `version`
- AND `servers` SHALL include at least one entry

### Requirement: Endpoint Coverage

The spec MUST document all 4 existing endpoints: `POST /shorten`, `GET /{slug}`, `GET /{slug}/stats`, and `GET /slugs`. Each endpoint SHALL include method, path, summary, and all response status codes.

#### Scenario: All endpoints are present

- GIVEN the spec file
- WHEN inspected
- THEN `paths` SHALL contain entries for `/shorten`, `/{slug}`, `/{slug}/stats`, and `/slugs`

#### Scenario: POST /shorten is fully documented

- GIVEN the `/shorten` path
- WHEN its POST operation is read
- THEN it SHALL define a request body requiring `url` (string)
- AND it SHALL document status codes `201`, `400`, and `500`

#### Scenario: GET /{slug} redirect is documented

- GIVEN the `/{slug}` path
- WHEN its GET operation is read
- THEN it SHALL define `slug` as a required path parameter
- AND it SHALL document `301` with a `Location` header, `404`, and `500`

#### Scenario: GET /{slug}/stats is documented

- GIVEN the `/{slug}/stats` path
- WHEN its GET operation is read
- THEN it SHALL document `200` with schema containing `original_url`, `click_count`, `created_at`
- AND it SHALL document `404` and `500`

#### Scenario: GET /slugs with pagination is documented

- GIVEN the `/slugs` path
- WHEN its GET operation is read
- THEN it SHALL define `page` (integer, default 1, min 1) and `limit` (integer, default 50, min 1, max 200)
- AND it SHALL document `200` with `data` array and `pagination` object, plus `400` and `500`

### Requirement: Schema Accuracy

Request and response schemas MUST reflect runtime JSON shapes. Types, required markers, and formats SHALL match actual behavior.

#### Scenario: Response schemas use correct types

- GIVEN any documented response schema
- WHEN compared to the runtime response
- THEN field names, types, and required markers SHALL match
- AND `created_at` SHALL use `format: date-time`
- AND `click_count` SHALL be `integer`

#### Scenario: Error schema is shared

- GIVEN any endpoint that can return an error
- WHEN the error schema is read
- THEN it SHALL define `{error: string}` and all error codes (400, 404, 500) SHALL `$ref` it

### Requirement: CORS Documentation

The spec SHALL document the `Location` header on responses that return it.

#### Scenario: Location header is documented

- GIVEN the `GET /{slug}` operation's `301` response
- WHEN the response is read
- THEN it SHALL include the `Location` response header

### Requirement: Spec Serving

The spec MUST be served at `GET /openapi.yaml` with `Content-Type: application/x-yaml`. The route SHALL be registered before wildcard routes to prevent conflicts.

#### Scenario: Spec is retrievable at runtime

- GIVEN the server is running
- WHEN a client sends `GET /openapi.yaml`
- THEN the response status SHALL be `200`
- AND the `Content-Type` header SHALL be `application/x-yaml`
- AND the response body SHALL be valid OpenAPI 3.1.0 YAML

#### Scenario: Spec route does not conflict with slug redirect

- GIVEN the router registers `/openapi.yaml` before `/{slug}`
- WHEN a request for `GET /openapi.yaml` arrives
- THEN it SHALL be handled by the spec endpoint, not the redirect handler
