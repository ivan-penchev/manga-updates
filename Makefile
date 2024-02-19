.PHONY: test build lint

APP_NAME ?= manga-updates

test:
	@echo "Running tests..."
	go test -v ./...

build:
	@echo "Building binary..."
	go build -o ./bin/$(APP_NAME) cmd/$(APP_NAME)/main.go

lint:
	@echo "Running linter..."
	golangci-lint run
