# URL Listing Specification

## Purpose

Expose all registered short codes with their metadata via a read-only endpoint.

## Requirements

### Requirement: List Slugs Endpoint

The system SHALL expose a `GET /slugs` endpoint that returns all registered short codes as a JSON array.

#### Scenario: Retrieve all slugs

- GIVEN three short codes exist in storage
- WHEN `GET /slugs` is called
- THEN the response status is 200
- AND the response body is a JSON array containing exactly three entries

#### Scenario: Empty database

- GIVEN no short codes exist in storage
- WHEN `GET /slugs` is called
- THEN the response status is 200
- AND the response body is an empty JSON array `[]`, not `null`

### Requirement: Slug Entry Fields

Each entry in the response array SHALL contain `slug`, `original_url`, `click_count`, and `created_at`.

#### Scenario: Entry contains all required fields

- GIVEN a short code exists with slug `abc12`, original URL `https://example.com`, 5 clicks, and a creation timestamp
- WHEN `GET /slugs` is called
- THEN the entry for `abc12` contains `slug` equal to `abc12`
- AND the entry contains `original_url` equal to `https://example.com`
- AND the entry contains `click_count` equal to 5
- AND the entry contains a non-empty `created_at` timestamp

### Requirement: Ordering

The response array SHALL be ordered by `created_at` in descending order (newest first).

#### Scenario: Newest slug appears first

- GIVEN three short codes created at different times
- WHEN `GET /slugs` is called
- THEN the first entry in the array is the most recently created slug
- AND the last entry is the oldest slug

### Requirement: Coexistence with Redirect

The `GET /slugs` route SHALL NOT interfere with the existing `GET /:slug` redirect behavior.

#### Scenario: Slugs endpoint returns listing even if slug named "slugs" exists

- GIVEN a short code with slug `slugs` exists in storage
- WHEN `GET /slugs` is called
- THEN the response is the slug listing (status 200 with JSON array), not a redirect for that short code

#### Scenario: Other slugs still redirect

- GIVEN a short code `abc12` exists in storage
- WHEN `GET /abc12` is called
- THEN the response is a 301 redirect to the original URL, unchanged from current behavior
