GOLANGCI_LINT_VERSION := v2.15.0
IMAGE ?= fairshare-api:dev

-include .env
export

.DEFAULT_GOAL := help
.PHONY: help run migrate test test-db cover lint build image

help: ## List targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-10s %s\n", $$1, $$2}'

run: ## Start the API using .env
	go run ./cmd/fairshare serve

migrate: ## Apply database migrations using .env
	go run ./cmd/fairshare migrate

test: ## Unit tests only; database tests skip
	TEST_DATABASE_URL= go test -race ./...

test-db: ## All tests, including those that need TEST_DATABASE_URL
	go test -race -count=1 ./...

cover: ## Coverage across packages, written to coverage.html
	go test -count=1 -coverpkg=./internal/...,./migrations -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -1

# The linter is built with the module's own toolchain; one built with an older
# Go refuses to analyse code that targets a newer one.
lint: ## golangci-lint, pinned to the CI version
	GOTOOLCHAIN=$$(go env GOVERSION) go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

build: ## Static binary in bin/
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/fairshare ./cmd/fairshare

image: ## Container image
	docker build -t $(IMAGE) .
