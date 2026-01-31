.PHONY: test build lint

APP_NAME ?= manga-updates

test:
	@echo "Running tests..."
	go test -v ./... -count=1

build:
	@echo "Building binary..."
	go build -o ./bin/$(APP_NAME) cmd/$(APP_NAME)/main.go

lint:
	@echo "Running linter..."
	golangci-lint run

mocks:
	@echo "Generating mocks..."
	go tool github.com/vektra/mockery/v3
