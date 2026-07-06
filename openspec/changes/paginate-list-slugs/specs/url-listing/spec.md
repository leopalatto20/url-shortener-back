# Delta for URL Listing

## MODIFIED Requirements

### Requirement: List Slugs Endpoint

The system SHALL expose a `GET /slugs` endpoint that returns registered short codes as a paginated JSON envelope containing `data` and `pagination` fields. The endpoint SHALL accept optional `page` and `limit` query parameters.

- `page` MUST default to `1`. Values less than `1` MUST be clamped to `1`.
- `limit` MUST default to `50`. Values less than `1` MUST be clamped to `50`. Values greater than `200` MUST be clamped to `200`.
- `pagination` MUST include `page`, `limit`, `total`, and `total_pages`.

(Previously: returned all slugs as a flat JSON array with no pagination)

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
- AND `data` is an empty JSON array `[]`
- AND `pagination.page` is `5`
- AND `pagination.total` is `10`

#### Scenario: Limit clamped above maximum

- GIVEN short codes exist in storage
- WHEN `GET /slugs?limit=999` is called
- THEN `pagination.limit` is `200`
- AND `data` contains at most 200 entries

#### Scenario: Limit clamped below minimum

- GIVEN short codes exist in storage
- WHEN `GET /slugs?limit=0` is called
- THEN `pagination.limit` is `50`

#### Scenario: Page clamped below minimum

- GIVEN short codes exist in storage
- WHEN `GET /slugs?page=0` is called
- THEN `pagination.page` is `1`

### Requirement: Slug Entry Fields

Each entry in the `data` array SHALL contain `slug`, `original_url`, `click_count`, and `created_at`.

(Previously: each entry was a direct element of the top-level response array)

#### Scenario: Entry contains all required fields

- GIVEN a short code exists with slug `abc12`, original URL `https://example.com`, 5 clicks, and a creation timestamp
- WHEN `GET /slugs` is called
- THEN `data[0]` contains `slug` equal to `abc12`
- AND `data[0]` contains `original_url` equal to `https://example.com`
- AND `data[0]` contains `click_count` equal to `5`
- AND `data[0]` contains a non-empty `created_at` timestamp

## ADDED Requirements

### Requirement: Error Wrapping

The service layer SHALL wrap all store-layer errors with contextual information before returning them to the handler. Raw store errors MUST NOT propagate unmodified.

#### Scenario: Store error is wrapped

- GIVEN the store returns an error when listing slugs
- WHEN the service processes the error
- THEN the returned error includes the original error as a wrapped cause
- AND the returned error includes context about the failed operation

## UNCHANGED Requirements

The following requirements from the main spec remain unchanged and are preserved as-is:

- **Ordering** — the `data` array SHALL be ordered by `created_at` descending (newest first).
- **Coexistence with Redirect** — `GET /slugs` SHALL NOT interfere with `GET /:slug` redirect behavior.
