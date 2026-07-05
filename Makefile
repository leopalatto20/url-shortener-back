.PHONY: build test run sqlc-generate clean

build:
	go build -o bin/url-shortener ./cmd/url-shortener

test:
	go test ./... -v -count=1

run:
	go run ./cmd/url-shortener

sqlc-generate:
	sqlc generate

clean:
	rm -rf bin/
	rm -f url-shortener.db