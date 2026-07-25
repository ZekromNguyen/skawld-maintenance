SHELL := /usr/bin/env bash
COMPOSE ?= podman compose
CONTAINER_ENGINE ?= podman

.PHONY: fmt vet test test-integration check build package migrate-up migrate-status compose-up compose-up-objectstore compose-down container-build

fmt:
	gofmt -w $$(find . -type f -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...

test:
	go test ./...

test-integration:
	TEST_DATABASE_URL="$${TEST_DATABASE_URL:?TEST_DATABASE_URL is required}" go test -race -count=1 ./...

check: fmt vet test

build:
	go build ./cmd/api ./cmd/worker ./cmd/migrate

package:
	go run ./tools/package -version "$${VERSION:-dev}" -clean

migrate-up:
	go run ./cmd/migrate up

migrate-status:
	go run ./cmd/migrate status

compose-up:
	$(COMPOSE) up -d --wait postgres keycloak

compose-up-objectstore:
	$(COMPOSE) --profile objectstore up -d --wait s3mock

compose-down:
	$(COMPOSE) down

container-build:
	$(CONTAINER_ENGINE) build -f deployments/containers/api.Dockerfile -t skawld-maintenance-api:dev .
	$(CONTAINER_ENGINE) build -f deployments/containers/worker.Dockerfile -t skawld-maintenance-worker:dev .
