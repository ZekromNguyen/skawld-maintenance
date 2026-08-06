# Design: Remaining Tasks After Backend Gap Completion

Date: 2026-08-06
Status: Approved
Branch: feature/maintenance-copilot-pilot-baseline

## Problem

The backend gap-completion plan (paginated lists, EAM/CMMS connector
operation, env-selectable AI providers) is implemented, reviewed, and merged.
The remaining work splits into four groups:

- **A. Engineering-closeable backlog**: deferred-minor items from the executed
  plan's reviews (OpenAPI drift, weak error-sentinel tests, missing negative
  tests), web console "load more" wiring for the six newly paginated lists,
  and a verify-and-document pass over the fictional demo corpus.
- **B. Deployment gates**: real AI provider contract tests (secret-gated) and
  the provider-selection runbook; transcription deployment documentation.
- **C. Native packaging + signing**: Flutter analysis/test on Windows/macOS CI
  runners, and a signing-gated release workflow (Authenticode + notarization).
- **D. Pilot-customer package**: a documented decision-and-drill checklist
  that stays blocked until a named pilot customer exists.

## Goals

- Close every engineering-closeable review item from the gap-completion plan
  that has real value; park the purely cosmetic ones explicitly.
- Make the six newly paginated console lists use the cursor API in the UI.
- Add secret-gated contract tests for the three AI HTTP adapters plus
  transcription, and the runbook to deploy a real provider.
- Run Flutter quality gates on the native CI runners and make release signing
  possible the moment signing secrets exist.
- Produce a pilot-customer decision/drill checklist that needs no code.

## Non-Goals

- Selecting or provisioning an actual AI provider, transcription endpoint,
  signing identity, or notarization account. This design makes those possible
  (tests, workflow steps, runbooks) but requires credentials from outside.
- Performance work: the N+1 loading in executions/documents/demonstrations/
  workflows list queries stays as-is (documented as future work).
- Changing the deterministic-provider default; CI and seed stay
  credential-free.
- A second implementation of anything already merged.

## A — Engineering-closeable backlog

### A1. OpenAPI drift fixes

`api/openapi.yaml` currently omits fields that are serialized
unconditionally:

- `Asset.created_at` (`internal/asset/application/service.go`, json
  `created_at`, no omitempty) — add to the `Asset` component schema.
- `Execution.updated_at` (`internal/execution/application/service.go`, json
  `updated_at`, no omitempty) — add to the `Execution` component schema.

Add the properties with `{type: string, format: date-time}` and add them to
the schema `required` lists where the Go struct always emits them.

### A2. Error-sentinel test hardening

Upgrade tests that only assert `err != nil` to assert the actual contract
sentinel so regressions that return the wrong error are caught:

- `internal/skawld/http_structured_provider_test.go` — empty content and
  non-JSON content: assert `errors.Is(err, ErrInvalidOutput)`.
- `internal/skawld/http_embedding_provider_test.go` — inconsistent
  dimensions: assert `errors.Is(err, ErrInvalidOutput)`.
- `internal/skawld/anthropic_structured_provider_test.go` — empty text and
  non-JSON text: assert `errors.Is(err, ErrInvalidOutput)`.
- `internal/platform/httpserver/integrations_test.go` — malformed snapshot
  and missing-attribute cases already assert 400; that status assertion is
  the contract (the internal `ErrInvalid` wrap is an implementation detail
  the handler tests need not observe). No change beyond confirming the
  400 assertions exist.

### A3. Missing negative/edge coverage

- Structured provider: a response body with `{"choices": []}` asserting
  `ErrInvalidOutput` (the guard the plan flagged).
- Anthropic provider: a response body with `"content": []` asserting
  `ErrInvalidOutput`.
- Embedding provider: data-count mismatch (server returns fewer embeddings
  than inputs), zero-length embedding, and non-2xx with status — all
  implemented but untested.
- List endpoints: for each of assets, incidents, executions, documents,
  demonstrations, workflows add handler tests for invalid `page_size` → 400,
  malformed cursor → 400, and a principal lacking the read permission → 403.
  (Reports/handovers already have these.)
- Store cursor-predicate integration tests (gated on `TEST_DATABASE_URL`) for
  assets, incidents, executions, documents, demonstrations — verifying
  `LIMIT page_size+1`, truncation, and `hasMore` against a real database.
  Reports/handovers/workflows already have coverage.
- `cmd/import` negative tests: missing `-snapshot`/`-site-id` returns an
  error; a non-administrator `-external-subject` returns the
  "no administrator principal" error; `-limit` out of range fails.

### A4. Web console "load more"

`web/src/api.ts` already exposes `fetchAll` (`api.ts:106`) and the
`ListPage<T> = ListResponse<T> & {next_cursor, has_more}` type
(`web/src/types.ts:356-358`). HandoverPage and ReportsPage already use it.
Wire it into the six remaining console list pages:

- AssetsPage, IncidentsPage, ExecutionsPage, KnowledgePage (documents),
  DemonstrationsPage, WorkflowsPage.

Each page: load the first page with `page_size` (e.g. 25), render a "Load
more" button when `has_more` is true, append the next page on click using
`next_cursor`. Shared behavior should live in one small hook or component
(`usePaginatedList`) rather than being copy-pasted six times; the six pages
may differ in loading/empty states but must share the cursor-walking logic.

### A5. Fictional corpus verification

`cmd/seed` already produces the Phase 2 corpus: an ingested and approved SOP,
incident history, recommendations with correction feedback, two reviewed
demonstrations, and per-role demo principals with scoped approval
authorities. Deliverable: a demo-data audit checklist (what the seed creates,
how to verify in the UI, cleanup SQL) documented in
`docs/runbooks/demo-data.md`, plus a `make seed`-driven verification step in
the demo runbook. No new corpus content is designed; gaps found during the
audit are reported back rather than silently designed around.

### A6. Explicitly parked

- N+1 loading in the four ID-then-load list queries.
- `has_more` derived from `next != ""` instead of a returned boolean
  (equivalent today; the service only emits a cursor when `hasMore`).
- `writeIntegrationImportError` duplicating `writeDomainError` shape.
- Config `Validate()` not normalizing provider values (Load and BuildProviders
  already normalize; a directly-constructed Config with padded casing would
  fail closed — acceptable).
- Sink trusting Importer-validated site/org scope (application layer
  validates first).

## B — Deployment gates

### B1. Secret-gated provider contract tests

New files in `internal/skawld`, each skipping unless its env vars are set
(following the repo's `TEST_DATABASE_URL`-style skip convention):

- `http_structured_provider_contract_test.go` — requires `AI_ENDPOINT`,
  `AI_API_KEY`, `AI_MODEL`, `AI_MODEL_VERSION`; performs one real
  `Generate` for `CapabilityRecommendation` with a fixed evidence payload and
  asserts: valid JSON output, non-empty `ProviderMetadata`, and that the
  output survives the closed recommendation decoder (unknown fields,
  invented evidence IDs, and over-advisory risk levels are rejected) without
  a decoder error.
- `http_embedding_provider_contract_test.go` — requires
  `EMBEDDING_ENDPOINT`, `AI_API_KEY`, `EMBEDDING_MODEL`,
  `EMBEDDING_MODEL_VERSION`; embeds two fixed texts and asserts consistent
  dimensions and COSINE metric.
- `anthropic_structured_provider_contract_test.go` — requires
  `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL`, `ANTHROPIC_MODEL_VERSION`; one real
  `Generate` for `CapabilityRecommendation`, same assertions.
- `http_transcription_provider_contract_test.go` — requires
  `TRANSCRIPTION_ENDPOINT`, `TRANSCRIPTION_API_KEY`, `TRANSCRIPTION_MODEL`,
  `TRANSCRIPTION_MODEL_VERSION`; transcribes a fixed silent audio fixture and
  asserts non-empty text or an explicit provider error.

CI: a `provider-contracts` job (or a step in `checks`) that runs
`go test ./internal/skawld/ -run Contract` with the secrets from
`secrets: inherit`-style configuration; the tests self-skip when the env is
absent, so the job is green without credentials and runs the contracts when
a repository has them configured.

### B2. Provider-selection runbook

`docs/runbooks/ai-providers.md`: documents `STRUCTURED_PROVIDER`,
`EMBEDDING_PROVIDER`, and credential env vars; endpoint formats for
OpenAI-compatible and Anthropic; the embedding-dimension re-ingestion warning
(switching `EMBEDDING_PROVIDER` after documents are embedded mixes vector
dimensions — must be paired with re-ingestion); and how to run the
secret-gated contract tests locally.

### B3. Transcription deployment

No code change. The transcription adapter is already env-gated
(`TRANSCRIPTION_ENDPOINT` etc.) and fails explicitly with 503 when
unavailable. The runbook documents the deployment choice (endpoint, model,
version) and how to verify transcripts (unverified candidates until a human
verifies or rejects).

## C — Native packaging and signing

### C1. Flutter quality gates on native runners

Extend the existing `desktop-app-packages` job in `.github/workflows/ci.yml`
(windows-latest and macos-latest runners) to run, before building:

- `dart format --output=none --set-exit-if-changed lib test`
- `dart run build_runner build`
- `flutter analyze`
- `flutter test`

This closes the "Flutter analysis/tests never ran on native runners" gate
from Plan.md Phase 2. The Linux `mobile` job already runs these; the native
runners add Windows/macOS coverage of the same checks.

### C2. Signing-gated release workflow

New `release.yml` (or steps in the existing packaging flow) that activate
only when the required secrets are present:

- Windows: Authenticode code signing of the packaged EXE/DLLs with a
  certificate imported from `WINDOWS_CERT_BASE64` + `WINDOWS_CERT_PASSWORD`,
  using the platform's `signtool`-equivalent via a pinned action.
- macOS: Developer ID Application signing with `MACOS_SIGNING_IDENTITY` and
  notarization via `MACOS_NOTARY_KEY_ID` / `MACOS_NOTARY_ISSUER_ID` /
  `MACOS_NOTARY_PRIVATE_KEY` (notarytool), with stapling.
- Steps use `if: ${{ secrets.XXX != '' }}` so the workflow is green before
  credentials exist and performs signing once they do.
- Add `docs/runbooks/desktop-signing.md`: how to provision the certificate
  and identities, where the secrets live, and a clean-machine install smoke
  test procedure (fresh VM install of the packaged app, launch, sign-in,
  load demo data).

## D — Pilot-customer package (documented checklist, blocked)

`docs/runbooks/pilot-readiness.md` documents, as a decision-and-drill
checklist that requires a named pilot customer before execution:

- Agreed topology and single-server/private deployment shape.
- Data classification and retention targets.
- IdP selection and the OIDC/SSO deployment guide steps to execute.
- S3/on-prem object-storage selection.
- Support contacts and escalation.
- RPO/RTO targets, then a restore drill measured against them
  (`scripts/restore-postgres.sh` + `scripts/verify-restore.sh`) and an
  object-storage backup/restore drill for the selected implementation.
- Safety review and customer acceptance sign-off.

No code changes; the deliverable is the runbook plus the existing drill
scripts being exercised once targets are agreed.

## Verification

- `gofmt -l .` empty; `go vet ./...` clean; `go build ./...` clean.
- `go test ./...` green; `TEST_DATABASE_URL=... go test -race -count=1 ./...`
  green (includes the new store cursor-predicate integration tests).
- `go run ./cmd/eval -dataset test/evaldata/pilot-v1.json` gates pass.
- OpenAPI validates (`python3 -c "import yaml; yaml.safe_load(...)"`).
- `cd web && npm run lint && npm test && npm run build` green (load-more
  wiring covered by the Vitest suite).
- CI: `checks`, `web`, `mobile`, `desktop-app-packages` (now with Flutter
  quality gates on native runners), and the signing-gated workflow all green
  without secrets.
- Contract tests skip cleanly without env credentials and run when provided.
