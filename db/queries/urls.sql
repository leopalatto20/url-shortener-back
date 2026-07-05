-- name: InsertUrl :execresult
INSERT INTO urls (slug, original_url)
VALUES (?, ?);

-- name: GetBySlug :one
SELECT original_url FROM urls
WHERE slug = ?;

-- name: IncrementClicks :exec
UPDATE urls SET click_count = click_count + 1
WHERE slug = ?;

-- name: GetStats :one
SELECT original_url, click_count, created_at FROM urls
WHERE slug = ?;

-- name: ListSlugs :many
SELECT slug, original_url, click_count, created_at FROM urls
ORDER BY created_at DESC;