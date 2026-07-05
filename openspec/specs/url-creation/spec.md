# URL Creation Specification

## Purpose

Accept a long URL, validate it, generate a unique short code, persist the mapping, and return the short URL.

## Requirements

### Requirement: URL Shortening Endpoint

The system SHALL expose a `POST /shorten` endpoint that accepts a JSON body with a `url` field and returns a shortened URL.

#### Scenario: Successful URL shortening

- GIVEN a valid HTTP(S) URL in the request body
- WHEN `POST /shorten` is called
- THEN the response status is 201
- AND the response body contains `short_code` (5-char alphanumeric) and `short_url` (full short URL)

#### Scenario: Invalid URL format

- GIVEN a request body with a malformed or non-HTTP URL
- WHEN `POST /shorten` is called
- THEN the response status is 400
- AND the response body contains an error message describing the validation failure

#### Scenario: Empty request body

- GIVEN an empty or missing request body
- WHEN `POST /shorten` is called
- THEN the response status is 400
- AND the response body contains an error message

### Requirement: Short Code Generation

The system SHALL generate a unique 5-character alphanumeric code for each shortened URL using a cryptographically secure random source.

#### Scenario: Auto-generated short code

- GIVEN a valid URL submitted for shortening
- WHEN the system generates a short code
- THEN the code is exactly 5 characters long
- AND the code contains only alphanumeric characters (`[a-zA-Z0-9]`)

#### Scenario: Collision handling

- GIVEN a generated short code that already exists in storage
- WHEN the system detects the collision on insert
- THEN the system SHALL retry with a new random code until a unique code is persisted

### Requirement: URL Persistence

The system SHALL persist the mapping between the generated short code and the original URL, along with a creation timestamp.

#### Scenario: URL mapping stored

- GIVEN a successful shortening request
- WHEN the short code is generated and validated
- THEN the system stores the short code, original URL, and `created_at` timestamp
- AND subsequent lookups by short code SHALL return the original URL
