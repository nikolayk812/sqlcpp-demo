.PHONY: generate test build

sqlc:
	sqlc generate

test:
	go test -v -race -cover ./...

build:
	go build ./...