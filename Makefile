SHELL := /usr/bin/env bash
COMPOSE ?= podman compose
CONTAINER_ENGINE ?= podman

.PHONY: fmt vet test test-integration check build eval sbom loadcheck package package-backend-desktop web-check mobile-check desktop-windows desktop-macos migrate-up migrate-status compose-up compose-resume compose-up-objectstore compose-down container-build pilot-config seed

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
	go build ./cmd/api ./cmd/worker ./cmd/migrate ./cmd/eval

eval:
	go run ./cmd/eval -dataset test/evaldata/pilot-v1.json -output dist/evaluation/pilot-v1.json

sbom:
	go run ./tools/sbom -version "$${VERSION:-dev}" -output dist/sbom/skawld-maintenance.cdx.json

loadcheck:
	go run ./tools/loadcheck -database-url "$${LOAD_DATABASE_URL:?LOAD_DATABASE_URL is required}"

package:
	go run ./tools/package -version "$${VERSION:-dev}" -clean

package-backend-desktop:
	go run ./tools/package -version "$${VERSION:-dev}" -platforms windows,darwin -clean

web-check:
	cd web && npm ci && npm run lint && npm test && npm run build

mobile-check:
	cd mobile && flutter pub get && dart run build_runner build && flutter analyze && flutter test

desktop-windows:
	cd mobile && flutter create --platforms=windows --org com.skawld --project-name skawld_maintenance_mobile . && flutter build windows --release
	cd mobile && pwsh ../scripts/package-desktop-windows.ps1 -Version "$${VERSION:-dev}"

desktop-macos:
	cd mobile && flutter create --platforms=macos --org com.skawld --project-name skawld_maintenance_mobile .
	cd mobile && ../scripts/configure-flutter-macos.sh
	cd mobile && flutter build macos --release
	cd mobile && ../scripts/package-desktop-macos.sh "$${VERSION:-dev}"

migrate-up:
	go run ./cmd/migrate up

migrate-status:
	go run ./cmd/migrate status

compose-up:
	$(COMPOSE) up -d --wait postgres keycloak

# Fast path for an already-provisioned stack (e.g. after a reboot or when the
# host was restarted): start the existing containers without the --wait health
# gate, config-diff checks, image pulls, or rebuilds. Containers should also
# auto-start at boot via the podman-restart user service (see ReadMe).
compose-resume:
	$(COMPOSE) start

compose-up-objectstore:
	$(COMPOSE) --profile objectstore up -d --wait s3mock

compose-down:
	$(COMPOSE) down

container-build:
	$(CONTAINER_ENGINE) build -f deployments/containers/api.Dockerfile -t skawld-maintenance-api:dev .
	$(CONTAINER_ENGINE) build -f deployments/containers/worker.Dockerfile -t skawld-maintenance-worker:dev .
	$(CONTAINER_ENGINE) build -f deployments/containers/migrate.Dockerfile -t skawld-maintenance-migrate:dev .
	$(CONTAINER_ENGINE) build -f deployments/containers/web.Dockerfile -t skawld-maintenance-web:dev .

pilot-config:
	$(COMPOSE) --env-file deployments/pilot/.env.example -f deployments/pilot/compose.yaml config

seed:
	go run ./cmd/seed
