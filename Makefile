.PHONY: build check clean fmt lint run sqlc-generate test

build:
	go build -o bin/url-shortener ./cmd/url-shortener

test:
	go test ./... -v -count=1

run:
	go run ./cmd/url-shortener

sqlc-generate:
	sqlc generate

fmt:
	gofumpt -l -w .

lint:
	golangci-lint run ./...

check: fmt lint test

clean:
	rm -rf bin/
	rm -f url-shortener.db