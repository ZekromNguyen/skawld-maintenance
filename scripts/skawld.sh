#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command_name="${1:-help}"
version="${2:-dev}"

cd "$repository_root"

load_environment() {
    if [[ ! -f .env ]]; then
        echo ".env is missing; copy .env.example to .env first" >&2
        exit 2
    fi
    set -a
    # .env is a developer-controlled configuration file.
    # shellcheck disable=SC1091
    source .env
    set +a
}

case "$command_name" in
    deps-up)
        docker compose up -d --wait postgres keycloak
        ;;
    deps-down)
        docker compose down
        ;;
    migrate)
        load_environment
        go run ./cmd/migrate up
        ;;
    api)
        load_environment
        go run ./cmd/api
        ;;
    worker)
        load_environment
        go run ./cmd/worker
        ;;
    check)
        while IFS= read -r -d '' go_file; do
            gofmt -w "$go_file"
        done < <(find cmd internal migrations tools -type f -name '*.go' -print0)
        go vet ./...
        go test ./...
        go build ./cmd/api ./cmd/worker ./cmd/migrate
        docker compose config --quiet
        ;;
    package)
        go run ./tools/package -version "$version" -clean
        ;;
    help|*)
        cat <<'EOF'
Usage: ./scripts/skawld.sh <command> [version]

Commands:
  deps-up             Start PostgreSQL and Keycloak through Docker Compose
  deps-down           Stop Compose dependencies
  migrate             Apply application and River migrations
  api                 Run the native Go API process
  worker              Run the native Go worker process
  check               Format, vet, test, build, and validate Compose
  package [version]   Build all Windows/macOS/Linux archives and checksums
EOF
        ;;
esac
