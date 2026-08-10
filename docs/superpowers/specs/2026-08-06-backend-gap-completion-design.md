# Design: Complete Remaining Backend Gaps

Date: 2026-08-06
Status: Approved
Branch: feature/maintenance-copilot-pilot-baseline

## Problem

A code review of the backend against `Plan.md` (Phases 0-5) found three
implemented-but-incomplete areas:

1. **AI provider deployment gate is open** — `cmd/api/main.go:125-130` and
   `cmd/worker/main.go` hard-wire `DeterministicProvider` /
   `DeterministicEmbeddingProvider` as the structured and embedding providers.
   Only transcription has a real HTTP adapter. Phase 2's deployment gate
   ("replace deterministic development providers with selected deployment
   adapters") cannot be closed.

2. **EAM/CMMS connector is unwired** — `internal/integration` implements the
   pull-only importer (`application/import.go`), the read connector contract
   (`domain/connector.go`), source-of-truth rules (`domain/source_of_truth.go`),
   and an NDJSON reader (`adapter/ndjson`), all with passing tests, but nothing
   references it from `cmd/` or the API. No `ProjectionSink` implementation
   exists, so a pilot operator cannot run an import.

3. **Unbounded list endpoints** — only `GET /reports` and `GET /handovers`
   paginate (`httpserver/reports.go:30`, `handovers.go:31`). The remaining six
   list endpoints return `{items}` with no `LIMIT`: assets, incidents,
   executions, documents, demonstrations, workflows. This was explicitly
   deferred by `2026-08-06-missing-list-apis-design.md` under Non-Goals.

## Goals

- Close the provider gate with a real, env-selected multi-provider layer while
  keeping deterministic providers as the default so CI/dev behavior is
  unchanged.
- Make the EAM/CMMS connector operable through both a CLI (`cmd/import`) and a
  protected API route, sharing one PostgreSQL projection sink.
- Standardize the six remaining list endpoints onto the existing keyset cursor
  envelope (`{items, next_cursor, has_more}`) with `page_size` and `cursor`
  query parameters.

## Non-Goals

- A connector marketplace or any write-back capability (explicitly forbidden
  by the Phase 5 acceptance criteria; `ReadConnector` stays pull-only).
- Automatic/scheduled connector pulls (River job). The chosen surface is
  operator-initiated (CLI) and on-demand (API). A River job can be added later
  without design changes.
- Replacing the deterministic evaluation suite, transcription provider, or
  vision provider. Those are unchanged.
- Web console "load more" UI wiring. Backend responses stay backward
  compatible so existing pages keep working; console pagination UI is a
  separate follow-up.

## Workstream 1 — Multi-provider AI layer

### Architecture

All adapters live in `internal/skawld` (the anti-corruption boundary owns the
provider contracts; no SDK leaks). Existing interfaces are unchanged:

- `StructuredProvider.Generate` — `routing.go:62`
- `EmbeddingProvider.Model` / `EmbeddingProvider.Embed` — `routing.go:118`
- `TranscriptionProvider` — `routing.go:130` (untouched)

New files:

- `internal/skawld/http_structured_provider.go` — `HTTPStructuredProvider`,
  OpenAI-compatible chat completions with JSON structured output
  (`response_format: {type: "json_object"}`). Config mirrors
  `http_transcription_provider.go`: endpoint, API key, provider, model, model
  version, HTTP client timeout.
- `internal/skawld/http_embedding_provider.go` — `HTTPEmbeddingProvider`,
  OpenAI-compatible embeddings API; returns a versioned `EmbeddingModel` and
  normalized vectors matching `DeterministicEmbeddingProvider` semantics.
- `internal/skawld/anthropic_structured_provider.go` —
  `AnthropicStructuredProvider`, Anthropic Messages API with JSON output.
  Anthropic has no embeddings API, so embeddings remain OpenAI-compatible or
  deterministic.

### Composition (env-driven selection)

`cmd/api/main.go`, `cmd/worker/main.go`, and `cmd/seed/main.go` build the
router/providers through one helper in `internal/skawld` (e.g.
`providers.FromConfig(cfg, httpClient)`).

Configuration follows the existing `config.go` conventions (unprefixed env
vars resolved through the `env()` helper, no `SKAWLD_` prefix). The `Config`
struct gains an `AI` section mirroring the existing `Transcription` section
(`config.go:86`), with validation at load time (`config.go:220-228`):

- `STRUCTURED_PROVIDER=deterministic|openai|anthropic` (default
  `deterministic`)
- `EMBEDDING_PROVIDER=deterministic|openai` (default `deterministic`)
- `AI_ENDPOINT`, `AI_API_KEY`, `AI_MODEL`, `AI_MODEL_VERSION` for the
  OpenAI-compatible adapters; `ANTHROPIC_API_KEY`/`ANTHROPIC_MODEL` for the
  Anthropic adapter; `EMBEDDING_ENDPOINT`/`EMBEDDING_MODEL` for embeddings.
- Any other provider value fails startup with an explicit error (fail
  closed). Missing credentials for a non-deterministic selection also fail
  startup, mirroring the transcription validation.
- Deterministic providers remain the fallback so CI, dev, and `cmd/seed` run
  without AI credentials.

### Error handling

- HTTP adapters validate status codes, bounded response bodies, JSON decoding
  of the configured output shape, and reject empty/malformed outputs with the
  same fail-closed contract the deterministic provider satisfies.
- Adapters never log API keys; config redaction mirrors the transcription
  provider.
- Timeouts are bounded by the configured HTTP client (default 30s for
  structured, 60s for embeddings, mirroring transcription's 4m only where
  transcription requires it).

### Testing

- Unit tests with `httptest` fakes covering: successful generation,
  malformed JSON, non-2xx status, empty output, timeout, and unknown provider
  config values (fail closed).
- Secret-gated contract tests (skip unless `AI_ENDPOINT`/`ANTHROPIC_API_KEY`
  and related env are set), following the existing
  `http_transcription_provider_test.go` pattern.
- Existing deterministic-provider tests must keep passing unchanged; the
  router contract tests in `internal/skawld/sdk_contract_test.go` remain the
  source of truth for provider output contracts.

## Workstream 2 — EAM/CMMS connector operation (CLI + API)

### Architecture

New PostgreSQL projection sink (shared by CLI and API):

- `internal/integration/adapter/postgres/sink.go` —
  `Sink{Pool, IDs, Clock, Audit}` implementing `application.ProjectionSink`
  (`Apply(ctx, principal, records)`).
- Maps `ExternalRecord` (kind `asset`) to `EXTERNAL_REFERENCE` rows in the
  `assets` table (migration `00002` already provides `source_of_truth`,
  `external_system`, `external_id`, `external_version`, and the unique
  constraint on `(organization_id, external_system, external_id)`).
- Rules, enforced per record before any write:
  - record.Validate (tenant/site scope) runs first; reject on violation;
  - an existing `OWNED_BY_SKAWLD` asset with the same external key is never
    modified or re-owned;
  - an existing `EXTERNAL_REFERENCE` asset is updated only when the incoming
    `external_version` differs (upsert by version, digest-checked);
  - unknown record kinds are rejected (no silent projection of
    work/permit/telemetry records).
- All writes occur in one transaction per page with audit events appended;
  a rejected page fails closed (no partial projection).
- Source-of-truth validation uses `integration/domain/source_of_truth.go`.

New CLI:

- `cmd/import/main.go` — flags: `-snapshot` (NDJSON file path, required),
  `-site-id` (required), `-database-url`, `-cursor` (optional resume),
  `-limit` (default 100, max 500, matching `Importer.Pull`).
- Loads an admin principal (must hold `PermissionExternalImport`) from the
  database, constructs `ndjson.Connector` over the snapshot, loops
  `Importer.Pull` until `Page.Complete`, and prints a per-page summary
  (`imported`, `rejected`, `next_cursor`) to stdout.
- Idempotent and re-runnable: cursor resume + version-based upserts make
  replay safe.

New API route:

- `POST /api/v1/integrations/imports` in `internal/platform/httpserver` —
  RBAC-gated on `PermissionExternalImport` (admin only), bounded multipart
  NDJSON upload (limit ~10 MB), streams to a temp file, runs the same
  `Importer.Pull` loop synchronously, returns
  `{imported, rejected, next_cursor, complete}`.
- Added to `api/openapi.yaml` with the error model and 403 for non-admin
  callers.
- Route mounting and dependency wiring in `cmd/api/main.go` +
  `httpserver/router.go` (`mountIntegrationRoutes`).

### Error handling

- Tenant/site scope violations abort the page and return a `Problem` (403 for
  forbidden, 400 for invalid records), never a partial write.
- CLI exits non-zero with a clear message on the first rejected page and
  prints the offending record kind/site.
- Multipart size and file-type checks reject non-NDJSON payloads before any
  DB work.

### Testing

- Sink integration tests against `TEST_DATABASE_URL` (style of
  `internal/execution/adapter/postgres/store_integration_test.go`):
  cross-tenant rejection, Skawld-owned preservation, external upsert by
  version, unknown-kind rejection, idempotent replay.
- CLI end-to-end: fixture NDJSON, temp DB, assert summary output and DB rows.
- API handler test with a fake sink (style of `router_test.go`):
  non-admin 403, oversized upload 413/400, happy path envelope.
- `ci.yml` gains the CLI e2e step (runs under the existing
  `TEST_DATABASE_URL` job); provider contract tests run when secrets are
  present and skip otherwise.

## Workstream 3 — Pagination on six list endpoints

### Architecture

Reuse the existing keyset pattern from reports/handovers
(`httpserver/maintenance_helpers.go:81-92` `parsePageSize`, cursor encoding
`created_at|id` cut at `strings.LastIndex(cursor, "|")` per
`handover/adapter/postgres/store.go:282-287`).

Per endpoint (assets, incidents, executions, documents, demonstrations,
workflows):

- Application filter gains `PageSize int` and `Cursor string`.
- Store query adds the cursor predicate (`(created_at, id) > (cursor)`
  ordering by `created_at, id`), `ORDER BY`, and `LIMIT page_size+1` to
  compute `has_more`.
- Handler reads `page_size` and `cursor`, returns
  `{items, next_cursor, has_more}`. `next_cursor` is null when the page is
  exhausted.
- Existing filters (`site_id`, `state`, `asset_id`, etc.) compose with
  pagination; site/org scoping is unchanged.
- OpenAPI adds `page_size` and `cursor` query params and the envelope fields
  to each of the six paths.

The response shape change (`{items}` to `{items, next_cursor, has_more}`) is
backward compatible: web consumers read `items`, and the new fields are
additive. Web "load more" wiring is explicitly out of scope (Non-Goals).

### Testing

- Store-level unit tests in the style of
  `internal/platform/httpserver/reports_list_test.go` and
  `handovers_list_test.go`: pagination walks, has_more correctness, cursor
  stability across inserts, filter+pagination composition.
- Handler tests asserting the envelope and 400 on invalid `page_size`.

## Verification

- `go build ./...`, `go vet ./...`, `go test ./...` pass.
- `make test-integration` (with `TEST_DATABASE_URL`) passes, including the
  new sink + CLI e2e tests.
- `make web-check` passes (unchanged web contract).
- OpenAPI validation passes with the new paths/params.
- `go run ./cmd/eval` still passes the frozen pilot safety gates.
- Manual smoke: run `cmd/import` against a fixture snapshot; call
  `POST /api/v1/integrations/imports` as admin and as non-admin.
