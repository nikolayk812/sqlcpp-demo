.PHONY: generate test build

sqlc:
	sqlc generate

test:
	TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock go test -v -race -cover ./...

build:
	go build ./...