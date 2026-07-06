# URL Listing Specification

## Purpose

Expose all registered short codes with their metadata via a read-only endpoint with pagination support.

## Requirements

### Requirement: List Slugs Endpoint

The system SHALL expose a `GET /slugs` endpoint that returns registered short codes as a paginated JSON envelope containing `data` and `pagination` fields. The endpoint SHALL accept optional `page` and `limit` query parameters.

- `page` MUST default to `1`. Values less than `1` or non-numeric values MUST be rejected with `400 Bad Request`. If `page` exceeds the last page, it is clamped to the last page.
- `limit` MUST default to `50`. Values less than `1`, greater than `200`, or non-numeric values MUST be rejected with `400 Bad Request`.
- `pagination` MUST include `page`, `limit`, `total`, and `total_pages`.

#### Scenario: Retrieve first page with defaults

- GIVEN 120 short codes exist in storage
- WHEN `GET /slugs` is called with no query parameters
- THEN the response status is 200
- AND the response body has `data` containing exactly 50 entries
- AND `pagination.page` is `1`
- AND `pagination.limit` is `50`
- AND `pagination.total` is `120`
- AND `pagination.total_pages` is `3`

#### Scenario: Retrieve a specific page

- GIVEN 120 short codes exist in storage
- WHEN `GET /slugs?page=3&limit=50` is called
- THEN `data` contains exactly 20 entries (the remaining items)
- AND `pagination.page` is `3`

#### Scenario: Empty database

- GIVEN no short codes exist in storage
- WHEN `GET /slugs` is called
- THEN the response status is 200
- AND `data` is an empty JSON array `[]`, not `null`
- AND `pagination.total` is `0`
- AND `pagination.total_pages` is `0`

#### Scenario: Page exceeds total pages

- GIVEN 10 short codes exist in storage
- WHEN `GET /slugs?page=5&limit=50` is called
- THEN the response status is 200
- AND `pagination.page` is clamped to `1` (the last page)
- AND `data` contains all 10 entries
- AND `pagination.total` is `10`
- AND `pagination.total_pages` is `1`

#### Scenario: Limit rejected below minimum

- GIVEN short codes exist in storage
- WHEN `GET /slugs?limit=0` is called
- THEN the response status is 400
- AND the response body contains `"invalid limit parameter"`

#### Scenario: Page rejected below minimum

- GIVEN short codes exist in storage
- WHEN `GET /slugs?page=0` is called
- THEN the response status is 400
- AND the response body contains `"invalid page parameter"`

#### Scenario: Limit rejected above maximum

- GIVEN short codes exist in storage
- WHEN `GET /slugs?limit=999` is called
- THEN the response status is 400
- AND the response body contains `"invalid limit parameter"`

### Requirement: Slug Entry Fields

Each entry in the `data` array SHALL contain `slug`, `original_url`, `click_count`, and `created_at`.

#### Scenario: Entry contains all required fields

- GIVEN a short code exists with slug `abc12`, original URL `https://example.com`, 5 clicks, and a creation timestamp
- WHEN `GET /slugs` is called
- THEN `data[0]` contains `slug` equal to `abc12`
- AND `data[0]` contains `original_url` equal to `https://example.com`
- AND `data[0]` contains `click_count` equal to 5
- AND `data[0]` contains a non-empty `created_at` timestamp

### Requirement: Ordering

The `data` array SHALL be ordered by `created_at` in descending order (newest first).

#### Scenario: Newest slug appears first

- GIVEN three short codes created at different times
- WHEN `GET /slugs` is called
- THEN the first entry in `data` is the most recently created slug
- AND the last entry in `data` is the oldest slug

### Requirement: Coexistence with Redirect

The `GET /slugs` route SHALL NOT interfere with the existing `GET /:slug` redirect behavior.

#### Scenario: Slugs endpoint returns listing even if slug named "slugs" exists

- GIVEN a short code with slug `slugs` exists in storage
- WHEN `GET /slugs` is called
- THEN the response is the slug listing (status 200 with JSON envelope), not a redirect for that short code

#### Scenario: Other slugs still redirect

- GIVEN a short code `abc12` exists in storage
- WHEN `GET /abc12` is called
- THEN the response is a 301 redirect to the original URL, unchanged from current behavior

### Requirement: Error Wrapping

The service layer SHALL wrap all store-layer errors with contextual information before returning them to the handler. Raw store errors MUST NOT propagate unmodified.

#### Scenario: Store error is wrapped

- GIVEN the store returns an error when listing slugs
- WHEN the service processes the error
- THEN the returned error includes the original error as a wrapped cause
- AND the returned error includes context about the failed operation
