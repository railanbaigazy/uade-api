.PHONY: run test lint

run:
	go run ./cmd/api

test:
	go test ./... -v -cover

format:
	go fmt ./...

lint:
	golangci-lint run ./...