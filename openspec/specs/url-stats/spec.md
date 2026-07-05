# URL Stats Specification

## Purpose

Return click count, creation timestamp, and original URL for a given short code.

## Requirements

### Requirement: Stats Endpoint

The system SHALL expose a `GET /:slug/stats` endpoint that returns usage statistics for a given short code.

#### Scenario: Successful stats retrieval

- GIVEN a short code that exists in storage
- WHEN `GET /:slug/stats` is called
- THEN the response status is 200
- AND the response body contains `original_url`, `click_count`, and `created_at`

#### Scenario: Unknown short code

- GIVEN a short code that does not exist in storage
- WHEN `GET /:slug/stats` is called
- THEN the response status is 404
- AND the response body contains an error message

### Requirement: Stats Accuracy

The returned `click_count` SHALL reflect the total number of successful redirects served for that short code.

#### Scenario: Click count reflects redirects

- GIVEN a short code that has been redirected 3 times
- WHEN `GET /:slug/stats` is called
- THEN `click_count` equals 3

#### Scenario: New URL has zero clicks

- GIVEN a newly created short code that has never been accessed via redirect
- WHEN `GET /:slug/stats` is called
- THEN `click_count` equals 0
