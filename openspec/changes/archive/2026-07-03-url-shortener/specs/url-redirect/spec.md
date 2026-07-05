# URL Redirect Specification

## Purpose

Resolve a short code to its original URL and issue a permanent redirect. Track each redirect as a click.

## Requirements

### Requirement: Redirect Endpoint

The system SHALL expose a `GET /:slug` endpoint that redirects to the original URL associated with the given short code.

#### Scenario: Successful redirect

- GIVEN a short code that exists in storage
- WHEN `GET /:slug` is called with that code
- THEN the response status is 301
- AND the `Location` header contains the original URL

#### Scenario: Unknown short code

- GIVEN a short code that does not exist in storage
- WHEN `GET /:slug` is called
- THEN the response status is 404
- AND the response body contains an error message

### Requirement: Click Tracking

The system SHALL increment a click counter each time a redirect is served for a valid short code.

#### Scenario: Click count incremented on redirect

- GIVEN a short code with click count N
- WHEN `GET /:slug` is called and a 301 redirect is issued
- THEN the stored click count for that short code becomes N+1

#### Scenario: No click increment on 404

- GIVEN an unknown short code
- WHEN `GET /:slug` is called and a 404 is returned
- THEN no click counter is modified
