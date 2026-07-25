# Skawld Maintenance Copilot — Architecture Specification

Status: **Accepted; Phase 0 foundation implementation active as of 2026-07-26**  
Date: 2026-07-26  
Decision owner: Product/engineering owner  
Implementation status: **Not started**  
Core principle: **Skawld should learn from normal industrial work while preserving safety, evidence, authority, and human control.**

This document is the architectural decision record for the first vertical product built on `skawld-sdk-go`. It is intentionally a product architecture, not an enterprise wish list and not a claim of compliance with any industrial standard.

## 1. Product Understanding

Skawld Maintenance Copilot is an **industrial maintenance intelligence layer**. Its primary product loop is:

```text
normal or exceptional maintenance work
        ↓
semantic events + context + evidence + decisions + outcomes
        ↓
demonstrations and corrections
        ↓
reviewable workflow candidates
        ↓
approved, versioned, applicable workflows
        ↓
evidence-backed technician guidance
        ↓
new execution evidence and human feedback
```

The product is not primarily a chat interface, a document search tool, a CMMS, a historian, a reliability suite, or an industrial control system. Those capabilities may be sources, destinations, or supporting features. The differentiator is preserving how experienced people make and execute maintenance decisions, then making that knowledge reusable without transferring authority to an AI model.

For small customers, Skawld provides the minimum native asset, incident, inspection, work record, and history capabilities required to run the intelligence loop. For enterprise customers, those records will often be projections or references synchronized from SAP PM, Maximo, Infor, Oracle, or another EAM/CMMS. Skawld should not compete to own inventory, purchasing, permits, schedules, high-frequency telemetry, or the authoritative asset register.

Maintenance execution and reliability engineering are related but separate bounded contexts:

- **Maintenance** records the problem, work, observations, measurements, decisions, actions, prerequisites, outcome, and report.
- **Reliability** asks why failures recur and evaluates failure modes, criticality, downtime, strategy, and effectiveness across time.
- V1 captures enough trustworthy maintenance facts to support later reliability analysis; it does not implement a reliability engineering suite.

The first proof is not “the AI diagnosed a pump.” It is:

> Given an asset, current measurements, approved documents, past incidents, and a validated procedure, Skawld can provide a traceable recommendation, capture the technician’s actual work, produce an approved report, and preserve corrections as evidence for future workflow improvement.

## 2. Architecture Constraints

1. `skawld-maintenance` depends on `skawld-sdk-go`; the SDK never imports maintenance code or maintenance vocabulary.
2. Domain rules remain deterministic and provider-independent. LLM, speech, vision, database, object storage, and external systems are adapters.
3. AI output is untrusted input. It cannot directly mutate authoritative state or invoke safety-significant actions.
4. Production workflows are immutable versions. Learning creates candidates; people review and publish them.
5. Every material recommendation is evidence-first and may legitimately return `INSUFFICIENT_EVIDENCE`.
6. The MVP is a modular monolith with one transactional database. Each additional service must pay for its operational cost.
7. Field writes are offline-first, idempotent, and store-and-forward. Attachment transfer is independent from structured record synchronization.
8. `organization_id` and `site_id` exist from day one on tenant-owned data; database-per-tenant does not.
9. Asset hierarchy is flexible but typed. Customers are not forced into a fixed number of levels.
10. Manually entered and low-frequency measurements may live in PostgreSQL. Raw high-frequency telemetry does not.
11. The application integrates with enterprise sources instead of becoming the owner of every enterprise process.
12. OT integration, if later required, is read-only by default through a separately deployed edge adapter and OT DMZ.
13. Public-cloud AI is optional. Provider contracts must allow private, OpenAI-compatible, and local endpoints.
14. Authentication is delegated through OIDC; authorization, approval authority, and audit semantics remain application responsibilities.
15. ISO 55001, ISO 14224, and ISA/IEC 62443 are mapping references, not compliance claims. Formal claims require domain, legal, and certification review.
16. The product monorepo contains backend, web, and mobile clients initially; the generic SDK stays in its own repository.
17. No direct PLC, DCS, SCADA, valve, motor, interlock, setpoint, shutdown, bypass, or safety-system control is in scope.
18. Every synchronized or projected record declares whether Skawld or an external system is the source of truth.
19. Maintenance domain/application packages do not import the SDK. All SDK types and upgrades are contained behind `internal/skawld`.
20. Role permissions and approval authority are distinct. A user may perform an action without being authorized to approve it.

## 3. Tech Stack Decision Matrix

Scores are 1 (poor) to 5 (excellent) for this product and present team, not universal technology rankings. “Offline/mobile integration” means suitability as the product backend and ability to support the mobile sync contract.

### 3.1 Backend

| Backend | Speed | Maintain | SDK fit | Types | AI ecosystem | Perf | Offline API | Deploy | On-prem | Test | Enterprise | Solo | Total / 60 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| **Go** | 4 | 5 | 5 | 5 | 3 | 5 | 4 | 5 | 5 | 5 | 5 | 4 | **55** |
| Python | 5 | 3 | 2 | 2 | 5 | 3 | 4 | 3 | 4 | 4 | 4 | 4 | 43 |
| TypeScript/Node.js | 5 | 4 | 2 | 4 | 4 | 4 | 5 | 4 | 4 | 4 | 4 | 5 | 49 |
| .NET | 3 | 5 | 2 | 5 | 3 | 5 | 4 | 4 | 5 | 5 | 5 | 3 | 49 |

Decision: **Go**. Calling the Go SDK from another runtime would require a network boundary, FFI, or duplicated abstractions before there is an independent scaling or team reason. Python remains an optional isolated document-processing worker only if measured extraction quality or required libraries justify it.

### 3.2 Application architecture

| Architecture | MVP speed | Boundary clarity | Operations | Independent scale | Enterprise path | Solo fit | Decision |
|---|---:|---:|---:|---:|---:|---:|---|
| **Modular monolith + separate worker process** | 5 | 4 | 5 | 3 | 4 | 5 | **Choose** |
| Microservices | 1 | 5 | 1 | 5 | 5 | 1 | Reject |
| Hybrid services from day one | 3 | 4 | 3 | 4 | 4 | 3 | Defer |

The API and worker are two process roles built from one codebase and share the same domain/application packages and transaction services. They do not call each other over HTTP and are not separate applications. That is a deployment split, not microservices:

```text
shared domain + application + adapters
                │
        ┌───────┴───────┐
        │               │
     API role       Worker role
```

### 3.3 Primary data store

| Store | Relational model | JSON | Search | Audit/transactions | Tenant path | Ops | Decision |
|---|---:|---:|---:|---:|---:|---:|---|
| **PostgreSQL** | 5 | 5 | 5 | 5 | 5 | 4 | **Choose** |
| SQLite | 4 | 3 | 2 | 3 | 2 | 5 | Mobile/dev only |
| MySQL | 5 | 4 | 3 | 5 | 5 | 4 | No advantage here |
| MongoDB | 2 | 5 | 3 | 3 | 4 | 3 | Poor domain fit |

PostgreSQL supports transactional domain data, append-oriented audit records, JSONB at uncertain boundaries, full-text search, row-level controls if later required, and pgvector without adding a database.

### 3.4 Semantic and text search

| Option | MVP operations | Metadata filtering | Hybrid search | Scale ceiling | Lock-in | Decision |
|---|---:|---:|---:|---:|---:|---|
| **PostgreSQL FTS + pgvector** | 5 | 5 | 5 | 3 | 5 | **Choose** |
| Qdrant | 3 | 4 | 3 | 5 | 4 | Defer |
| Weaviate | 2 | 4 | 4 | 5 | 3 | Defer |
| Pinecone | 4 | 4 | 3 | 5 | 1 | Reject for on-prem |
| OpenSearch/Elasticsearch | 1 | 4 | 5 | 5 | 3 | Defer |

Start with exact vector search and metadata filters. MVP hybrid retrieval runs two eligible-set searches—PostgreSQL FTS and cosine vector search—then combines ranked lists with Reciprocal Rank Fusion (RRF). There is no implicit `hybrid_search()` facility and no reranker model in V1.

Add HNSW only after query/load measurements justify it. With approximate indexes, confirm recall under tenant/site/applicability filters; evaluate iterative scans, partial indexes, or partitioning when the filtered candidate set is too sparse. Authorization and applicability are query eligibility predicates, never AI post-processing.

Graduate to dedicated vector/search infrastructure when one or more are true: the vector working set no longer fits the PostgreSQL performance envelope, indexing harms transactional workloads, independent search scaling is required, hybrid ranking requirements exceed PostgreSQL capabilities, or measured latency/recall SLOs cannot be met after normal tuning.

### 3.5 Web

| Web option | Dashboard ecosystem | Type safety | Static/on-prem | Server duplication risk | Maintainability | Solo fit | Decision |
|---|---:|---:|---:|---:|---:|---:|---|
| **React + TypeScript + Vite** | 5 | 5 | 5 | 5 | 5 | 5 | **Choose** |
| Next.js | 5 | 5 | 3 | 2 | 4 | 4 | Reject server runtime for V1 |
| Vue | 4 | 5 | 5 | 5 | 4 | 5 | Viable, smaller team pool |
| Svelte | 3 | 4 | 5 | 5 | 3 | 5 | Ecosystem risk |

Next.js does not provide material value for an authenticated operations dashboard that needs no SEO or server rendering. A static React build consumes the Go REST API and can be served by a reverse proxy or embedded later. Business logic and authorization stay in Go.

### 3.6 Mobile

| Mobile option | Android | Offline | Camera/voice | iOS path | Existing skill | Industrial UX | Decision |
|---|---:|---:|---:|---:|---:|---:|---|
| **Flutter** | 5 | 5 | 5 | 5 | 5 | 5 | **Choose** |
| React Native | 5 | 4 | 5 | 5 | 3 | 4 | Viable |
| Native Android | 5 | 5 | 5 | 1 | 3 | 5 | Only for Android-exclusive future |
| PWA | 3 | 3 | 3 | 4 | 4 | 3 | Insufficient field reliability |

Flutter with Drift over SQLite is the most credible single-team route to Android-first, offline-first, camera/voice-capable software and preserves an iOS path.

## 4. Recommended Stack

### 4.1 Backend

- **Choice:** Go, matching a supported Go version of `skawld-sdk-go`; `net/http` with `chi`, `pgx`, and `sqlc` where stable queries benefit from generation.
- **Why:** In-process SDK integration through a thin `internal/skawld` boundary, strong concurrency and cancellation, low resource use, static deployment, explicit types, good tests, and strong on-prem fit.
- **Alternatives considered:** Python, TypeScript/Node.js, .NET.
- **Why rejected:** They add an inter-process SDK boundary or duplicate the workflow runtime without a compensating advantage. Python’s AI ecosystem alone is not enough.
- **When to reconsider:** A genuinely independent ML/document service has unique libraries, scaling, hardware, or team ownership requirements.

### 4.2 Architecture

- **Choice:** Domain-oriented modular monolith; API and worker are separately runnable binaries from one codebase.
- **Why:** Preserves transaction boundaries and refactoring speed while giving ingestion/AI jobs independent process limits.
- **Alternatives considered:** Microservices and an early hybrid.
- **Why rejected:** Network contracts, deployment, tracing, eventual consistency, and failure modes would dominate a solo MVP.
- **When to reconsider:** A module has a distinct team, security zone, scaling pattern, release cadence, or hard runtime dependency.

Initial modules are `asset`, `incident`, `execution` (inspection and maintenance work), `knowledge`, `workflow`, `handover`, `identity`, `integration`, `skawld`, and `platform`. A later `reliability` bounded context consumes verified maintenance facts without owning execution. “Work order” is an external projection/reference or lightweight coordination record, not a full planning module.

### 4.3 Database

- **Choice:** PostgreSQL as the only server database.
- **Why:** The domain is relational; transactions, JSONB, full-text search, pgvector, audit, and later tenant controls coexist well.
- **Alternatives considered:** SQLite, MySQL, MongoDB.
- **Why rejected:** SQLite lacks the target concurrency/search posture; MySQL adds no benefit; MongoDB weakens relational integrity and reporting.
- **When to reconsider:** Never casually. Add TimescaleDB or historian integration only for measured high-volume time-series requirements; keep business records in PostgreSQL.

### 4.4 Vector search

- **Choice:** pgvector in PostgreSQL, initially exact cosine search with strict applicability/authority filters.
- **Why:** One backup, one security boundary, transactional metadata, and adequate early scale.
- **Alternatives considered:** Qdrant, Weaviate, Pinecone.
- **Why rejected:** They add operations and consistency work before scale requires it; Pinecone conflicts with restricted/on-prem deployments.
- **When to reconsider:** Search load affects OLTP, corpus/index size misses SLOs after tuning, or independent vector scaling becomes necessary.

### 4.5 Search

- **Choice:** PostgreSQL full-text search plus exact pgvector cosine search, with explicit RRF fusion in application code and no model reranker in V1.
- **Why:** Asset tags, exact terms, document identifiers, and error codes need lexical search; narrative similarity needs embeddings. Authorization, tenant, site, validity, authority, and applicability define the eligible set before either ranked search runs.
- **Alternatives considered:** OpenSearch/Elasticsearch.
- **Why rejected:** A search cluster is unjustified for the V1 corpus and team.
- **When to reconsider:** Complex multilingual analysis, highlighting/faceting, log-scale corpus size, or independent search SLOs demonstrably exceed PostgreSQL.

### 4.6 Cache

- **Choice:** No Redis in V1; bounded in-process caches only for immutable/reference data with explicit TTLs.
- **Why:** The MVP does not need distributed sessions, locks, pub/sub, or a high-rate shared cache.
- **Alternatives considered:** Redis/Valkey.
- **Why rejected:** Another stateful service, cache invalidation, and failure mode without a measured need.
- **When to reconsider:** Multiple API replicas require shared ephemeral coordination, distributed rate limits/locks are proven necessary, or measured hot reads justify it. Prefer database or stateless designs first.

### 4.7 Async jobs

- **Choice:** River-backed PostgreSQL jobs executed by `cmd/worker`; transactional enqueue, a dedicated worker PostgreSQL pool, bounded per-job-type concurrency, deadlines, retry classification, and idempotent workers.
- **Why:** Durable ingestion, embedding, transcription, vision, report, and notification work without a second datastore. A mature job library is safer than casually rebuilding leases and retries.
- **Alternatives considered:** goroutines only, custom `SKIP LOCKED` table, Redis queues, RabbitMQ, NATS, Kafka.
- **Why rejected:** Goroutines are not durable; a custom queue is deceptively complex; brokers are premature. Kafka is explicitly inappropriate.
- **When to reconsider:** PostgreSQL job connections, CPU, I/O, locks, or latency measurably harm OLTP despite dedicated pools and bounded concurrency; or cross-language producers, very high throughput, or independent broker semantics are required.

Initial concurrency is configuration, not a promise: start conservatively (for example embedding 5, report 10, vision 3, transcription 3), measure on the deployment hardware, and provide global connection-budget protection. Every job type declares timeout, maximum attempts, retryable/permanent error classification, idempotency key, and payload size limit.

### 4.8 API

- **Choice:** Versioned JSON REST (`/api/v1`) with OpenAPI 3, externally; Go application interfaces/commands internally.
- **Why:** Works for web, Flutter, external integrations, idempotent sync, debugging, and enterprise gateways.
- **Alternatives considered:** GraphQL and gRPC.
- **Why rejected:** GraphQL adds authorization/query complexity; gRPC is less convenient for first-party mobile/web and external adopters.
- **When to reconsider:** gRPC for a controlled edge link or internal high-throughput stream; GraphQL only if client query diversity becomes a measured problem.

### 4.9 Web

- **Choice:** React + TypeScript + Vite, TanStack Query, TanStack Router, TanStack Table, and generated API types from OpenAPI.
- **Why:** Strong dashboard/forms/tables ecosystem, static output, no duplicated business backend, and broad maintainability. The browser uses an OIDC-backed Go session with `Secure`, `HttpOnly`, appropriately scoped `SameSite` cookies; access/refresh tokens are not stored in `localStorage`.
- **Alternatives considered:** Next.js, Vue, Svelte.
- **Why rejected:** Next’s server features are unnecessary; Vue is viable but offers no product-specific advantage; Svelte has a smaller enterprise component/tooling pool.
- **When to reconsider:** Use Next.js only if a real server-rendered public surface or justified BFF requirement appears. Never move domain rules into it.

### 4.10 UI

- **Choice:** Tailwind CSS + Radix primitives + a curated local shadcn/ui component set; TanStack Table/Form; accessible charts used sparingly.
- **Why:** Provides control for dense industrial layouts without a proprietary component runtime. Components live in the repository and can be hardened for glove use, contrast, status, and reduced motion.
- **Alternatives considered:** Material UI and Ant Design.
- **Why rejected:** Both are productive but impose stronger visual/runtime systems and make a distinct, density-calibrated field product harder to tune.
- **When to reconsider:** A large team needs a fully governed design system or a customer mandates an existing enterprise component library.

Design rules: neutral palette, no decorative gradients/glass, compact tables, explicit severity and sync state, persistent context, readable timelines, keyboard support, WCAG-minded contrast, large mobile targets, and no animation required to understand state.

### 4.11 Mobile

- **Choice:** Flutter, Android first, iOS-compatible architecture.
- **Why:** Existing team experience, reliable native plugins, controlled rendering, and a credible offline database story.
- **Alternatives considered:** React Native, native Android, PWA.
- **Why rejected:** React Native has no decisive advantage; native Android closes the iOS path; PWA background sync/media behavior is not reliable enough for the primary field client.
- **When to reconsider:** Regulated device fleets require native platform controls or Flutter support becomes a documented blocker.

### 4.12 Mobile database

- **Choice:** SQLite through Drift.
- **Why:** Transactions, migrations, relational queries, reactive views, testability, and explicit outbox modeling.
- **Alternatives considered:** Isar and raw SQLite.
- **Why rejected:** Isar introduces another data model; raw SQLite creates avoidable mapping/migration work.
- **When to reconsider:** Only if measured device performance or a platform constraint proves Drift unsuitable.

### 4.13 Object storage

- **Choice:** Application-owned capability-level `ObjectStore` port implemented by an S3 adapter; no server product is the architectural default. The V1 port exposes only put/get/delete/head and signed upload/download operations required by Skawld.
- **Why:** Separates binary durability from PostgreSQL while avoiding assumptions that every “S3-compatible” server supports the complete S3 surface identically. Checksums, quarantine state, and authorization remain application concerns.
- **Alternatives considered:** Local filesystem, MinIO Community, direct cloud-specific APIs.
- **Why rejected:** Filesystem is not horizontally safe and complicates backup; cloud-specific APIs harm portability. MinIO Community is no longer the default because its upstream repository was archived in April 2026 and its AGPL/source-only posture requires explicit legal and support review.
- **When to reconsider:** Add multipart upload, object versioning, retention/object lock, or other capabilities only when a real file-size, compliance, or lifecycle requirement exists. Select a concrete product per deployment after security, license, support, required-operation compatibility, backup, and restore evaluation.

Conceptual V1 port:

```go
type ObjectStore interface {
    Put(ctx context.Context, key string, body io.Reader, size int64, metadata ObjectMetadata) (ObjectInfo, error)
    Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)
    Delete(ctx context.Context, key string) error
    Head(ctx context.Context, key string) (ObjectInfo, error)
    SignedUploadURL(ctx context.Context, key string, constraints UploadConstraints) (SignedURL, error)
    SignedDownloadURL(ctx context.Context, key string, expires time.Duration) (SignedURL, error)
}
```

This is a product capability contract, not a claim to abstract all of Amazon S3.

### 4.14 AI providers

- **Choice:** Use `skawld-sdk-go` provider abstractions for agent runtime, with maintenance-owned capability routing (`extract`, `reason`, `vision`, `speech`, `embed`) and deployment configuration.
- **Why:** Keeps model/provider names out of domain code and allows public, private, compatible, or local endpoints.
- **Alternatives considered:** Direct OpenAI/Anthropic SDK calls throughout the application; a separate AI gateway on day one.
- **Why rejected:** Direct calls create lock-in; a gateway adds a service before routing volume warrants it.
- **When to reconsider:** Add a gateway when multiple applications need centralized quotas, redaction, routing, and provider governance.

### 4.15 Embeddings

- **Choice:** Maintenance `EmbeddingProvider` port with model key, provider, model/version, dimensions, distance metric, input hash, created time, and indexing status stored per embedding.
- **Why:** Supports similar incidents and document chunks, reproducibility, and controlled re-embedding without leaking dimensions into domain objects.
- **Alternatives considered:** Provider-specific embedding code and a single unversioned vector column.
- **Why rejected:** Both prevent safe upgrades and provenance.
- **When to reconsider:** Introduce a dedicated embedding service when batching/GPU utilization or multiple products justify it.

### 4.16 Speech

- **Choice:** `TranscriptionProvider` port; cloud provider first for speed, private/local Whisper-compatible adapter for restricted deployments. Store original audio, transcript version, language, timestamps when available, model metadata, and verification state.
- **Why:** Voice is high value but deployment connectivity and privacy vary.
- **Alternatives considered:** Cloud-only speech and mandatory local inference.
- **Why rejected:** Cloud-only blocks air-gapped paths; local-only burdens the MVP.
- **When to reconsider:** Pilot privacy, latency, language, or cost data selects the default deployment adapter.

### 4.17 Vision

- **Choice:** `VisionProvider` returns schema-validated `ObservationCandidate[]`; a human must verify or reject each candidate before it becomes an `Observation`.
- **Why:** Visual inference is useful evidence assistance, not equipment authority.
- **Alternatives considered:** Free-text image diagnosis or automatic observation creation.
- **Why rejected:** Both erase uncertainty and create an unsafe authority path.
- **When to reconsider:** Never remove human verification for safety-relevant observations; only tune which low-risk metadata can be auto-accepted.

### 4.18 Authentication

- **Choice:** OIDC through an external identity provider. Web uses an authorization-code flow terminated by Go and a short-lived server-side session identified by a `Secure`, `HttpOnly`, appropriately scoped `SameSite` cookie; Flutter uses Authorization Code + PKCE and stores credentials only in platform secure storage. Development uses a containerized local OIDC provider.
- **Why:** Avoids creating password reset, MFA, account recovery, and SSO machinery while fitting enterprise and on-prem identity.
- **Alternatives considered:** Custom JWT/password IAM, SaaS-only identity, bespoke sessions only.
- **Why rejected:** Custom IAM creates security scope; SaaS-only identity blocks restricted deployments; session-only does not serve mobile/integration clients cleanly.
- **When to reconsider:** The concrete IdP is a deployment choice; the OIDC boundary remains. Add SCIM/SAML federation only for customer requirements.

### 4.19 Authorization

- **Choice:** Application-owned RBAC for action permissions plus a separate `ApprovalAuthority` model. Roles begin as Technician, Senior Technician, Supervisor, Manager, Administrator; permissions are named capabilities, not role checks scattered in handlers.
- **Why:** Being allowed to perform an action is not the same as being authorized to approve a workflow or safety-relevant decision. Approval authority is explicitly scoped by organization/site, subject, workflow/asset/asset class or competency, risk ceiling, delegation, and validity window.
- **Alternatives considered:** Fine-grained ABAC and IdP-role-only checks.
- **Why rejected:** ABAC is premature; IdP roles cannot express resource scope, workflow prerequisites, or approval separation.
- **When to reconsider:** Keep authorization RBAC-based for MVP. Evolve approval-policy evaluation only when real cases require richer conditions; do not introduce a general ABAC engine preemptively.

### 4.20 Testing

- **Choice:** Go `testing`, table tests, `testify` only where useful, Testcontainers for PostgreSQL/S3 integration, HTTP contract tests, Flutter unit/widget/integration tests, Vitest + Testing Library + Playwright for web.
- **Why:** Fast deterministic tests plus realistic adapter verification.
- **Alternatives considered:** Heavy mocking frameworks and live-model tests in the normal suite.
- **Why rejected:** Generated mocks hide contracts; live AI calls are slow, costly, and nondeterministic.
- **When to reconsider:** Add specialized load, chaos, device-farm, and model evaluation tooling at pilot scale.

Required fakes: `FakeLLM`, `FakeEmbeddingProvider`, `FakeVisionProvider`, `FakeTranscriptionProvider`, `FakeClock`, `FakeObjectStore`, `FakeIdentity`, and in-memory maintenance repositories. Provider contract tests are opt-in and secret-gated.

### 4.21 Observability

- **Choice:** Structured JSON logs via `slog`, request/correlation IDs, domain IDs, Prometheus-compatible metrics, and OpenTelemetry APIs wired initially to logs/local collector as needed.
- **Why:** Correlation is essential; a full distributed tracing backend is not.
- **Alternatives considered:** Vendor-specific agents or an immediate observability stack.
- **Why rejected:** Lock-in and operational weight before there are distributed services.
- **When to reconsider:** Add an OTLP collector/backend when multiple processes/deployments or pilot SLO diagnosis make traces materially useful.

Audit is separate: append-oriented, application-defined business/security records with actor, tenant, site, action, entity, before/after or patch, reason, request/execution/workflow/approval IDs, AI involvement, provenance, and timestamp. Application logs may be deleted on an operations schedule; audit retention follows policy and cannot depend on log scraping.

### 4.22 Containers

- **Choice:** OCI images for API, worker, and static web; Compose-compatible local/pilot deployment. PostgreSQL, OIDC dev provider, and optional S3 emulator are dependencies. Run Go/web/Flutter natively during normal development.
- **Why:** Works with Fedora Podman or Docker and supports VPS/on-prem without local Kubernetes.
- **Alternatives considered:** Kubernetes-first and all-development-in-containers.
- **Why rejected:** Both slow the solo feedback loop and add infrastructure unrelated to product proof.
- **When to reconsider:** Use Kubernetes only when the actual target platform, operations team, HA/SLO, or fleet deployment requires it.

### 4.23 CI/CD

- **Choice:** GitHub Actions for formatting/lint, unit tests, PostgreSQL integration tests, web/Flutter checks, dependency/security scanning, SBOM, and image build on protected branches/tags. No automatic production deployment yet.
- **Why:** Enforces quality and creates deployable artifacts without inventing a target environment.
- **Alternatives considered:** Complex promotion/GitOps pipelines.
- **Why rejected:** There is no production topology or change authority to automate.
- **When to reconsider:** Add signed artifacts, provenance, staged deployment, rollback, and environment approvals when a pilot target is selected.

### 4.24 Document processing

- **Choice:** Go orchestration and simple extraction/chunking adapters in the worker; an optional isolated Python extractor only after a representative-document bake-off proves a material quality/library advantage.
- **Why:** Parsing is an untrusted ingestion boundary, not domain logic. Go can own lifecycle, limits, provenance, retries, and indexing while the extractor remains replaceable.
- **Alternatives considered:** Python service from day one and all parsing inside the API process.
- **Why rejected:** Python adds a runtime/service without evidence; parsing in the API couples slow, hostile input to request availability.
- **When to reconsider:** Scanned PDFs, complex layout/tables/OCR, proprietary formats, or GPU pipelines show materially better evaluated results through a Python-only stack. Run it as a constrained worker, not a new business service.

## 5. System Architecture

```text
 APPROVED FIELD DEVICE                       SUPERVISOR / REVIEWER
 Flutter + Drift + outbox                    React static web
          │ HTTPS / REST / idempotency              │
          └──────────────────┬───────────────────────┘
                             ▼
┌──────────────────────────────────────────────────────────────────────┐
│                    SKAWLD MAINTENANCE (Go)                           │
│                                                                      │
│  HTTP adapters ── application commands/queries ── authorization      │
│                         │                                            │
│  ┌────────┬─────────┬───┴────┬──────────┬──────────┬──────────────┐  │
│  │ Asset  │Incident │Execution│Knowledge │ Workflow │  Handover    │  │
│  └────────┴─────────┴───┬────┴──────────┴──────────┴──────────────┘  │
│                         │                                            │
│     internal/skawld integration boundary (the only SDK importer)     │
│       tools / observers / stores / policies / model routing          │
│                         ▼                                            │
│             skawld-sdk-go (external dependency)                      │
│       agent | workflow | observation | learning | policy | audit     │
│                                                                      │
│  integration adapters             platform adapters                  │
│  EAM/CMMS | document | edge       PostgreSQL | S3 | OIDC | AI        │
└──────────────┬──────────────────────────────┬────────────────────────┘
               │                              │
       ┌───────┴────────┐            ┌────────┼───────────┐
       ▼                ▼            ▼        ▼           ▼
 PostgreSQL+pgvector   S3 target   LLM/AI   OIDC IdP   API/worker
 domain/search/jobs    binary data providers            telemetry

 ENTERPRISE IT                                   OT (future, separate)
 SAP/Maximo/CMMS ── adapter/API            PLC/DCS/SCADA/Historian
                                                      │ read-only
                                                      ▼
                                              Go edge gateway
                                                      │
                                                   OT DMZ
                                                      │ outbound,
                                                      ▼ store-forward
                                                    Skawld
```

Runtime rules:

- API and worker are stateless with respect to local disk.
- API and worker are independently scalable roles of the same application. They share internal packages and database transaction semantics; there is no API-to-worker HTTP contract.
- PostgreSQL is the transaction boundary for business state, outbox/inbox deduplication, audit, search metadata, and jobs.
- API and worker use separate, explicitly budgeted PostgreSQL pools. Worker queues enforce job-type concurrency and deadlines so async load cannot consume the API connection budget.
- Objects are uploaded through presigned URLs where deployment permits. A structured attachment record is committed separately and tracks upload/scan states.
- Enterprise adapters translate external identities and records into Skawld’s canonical application contracts while retaining source IDs and provenance.
- An edge gateway is not part of V1. It never exposes arbitrary industrial protocol tools to the agent and is read-only unless a future, separately governed product decision says otherwise.

Deployment profiles share the same application contracts:

- **Cloud/VPS:** managed PostgreSQL/S3/OIDC and configured public or private AI endpoints.
- **Private/on-premise:** customer-supported PostgreSQL, S3-compatible storage, OIDC, and optional private AI endpoint.
- **Restricted/air-gapped (future):** offline image/model/document bundles, local OIDC/S3/embeddings/AI, controlled export/import, and no permanent public API dependency.

V1 validates cloud/single-server portability but does not promise every profile. Provider clients must not assume public DNS, public model catalogs, or runtime downloads.

## 6. Boundary With `skawld-sdk-go`

### 6.1 Dependency direction

```text
maintenance domain/application
          │ no SDK types
          ▼
internal/skawld integration ports/adapters
          │ only product package allowed to import
          ▼
github.com/ZekromNguyen/skawld-sdk-go

skawld-sdk-go must never import skawld-maintenance.
```

Compile-time import guard tests enforce both directions. An SDK upgrade should normally change `internal/skawld`, its contract tests, and composition—not asset, incident, execution, knowledge, handover, or reliability business objects.

### 6.2 Ownership

| `skawld-sdk-go` owns (generic) | `skawld-maintenance` owns (industrial maintenance) |
|---|---|
| Agent/session/provider/tool contracts | Asset, hierarchy, component, criticality |
| Tool registry and generic descriptors | Incident, inspection/execution, work reference |
| Deterministic workflow model/executor | Measurement, engineering units, data quality |
| Semantic demonstration/event/trace model | Maintenance semantic event taxonomy/adapters |
| Multi-demonstration analysis and candidate compiler | Workflow applicability and maintenance review UI |
| Generic evidence references | Evidence authority, document validity, asset applicability |
| Policy/approval contracts and execution checkpoints | Maintenance risk classification and approval authority |
| Generic audit event/sink contracts | Business audit persistence, retention, display |
| Generic workflow/version/candidate stores | PostgreSQL implementations and tenant scoping |
| Provider-independent learning extractor boundary | Maintenance prompts/schemas/extractors |
| Generic human correction representation | Maintenance correction context and outcome |

The SDK already has `workflow`, `observation`, `learning`, `policy`, and `audit` packages. `internal/skawld` adapts them rather than building parallel runtimes or leaking their types into the maintenance domain. The SDK currently ships SQLite stores; product integration adapters implement the SDK store interfaces over PostgreSQL where required.

### 6.3 Maintenance adapters and tools

Read tools:

```text
maintenance.get_asset_context
maintenance.get_maintenance_history
maintenance.search_similar_incidents
maintenance.search_approved_knowledge
maintenance.get_workflow
maintenance.get_inspection_context
```

Write/proposal tools:

```text
maintenance.record_observation
maintenance.record_measurement
maintenance.prepare_report_draft
maintenance.create_work_reference_draft
maintenance.submit_inspection
maintenance.capture_decision
```

Tool names and schemas are versioned contracts. Each implementation:

1. authenticates a principal supplied by the application, never by model text;
2. rechecks organization/site/resource authorization;
3. parses a strict schema and rejects unknown/invalid fields;
4. performs deterministic domain/workflow validation;
5. declares read/write scope, parallel safety, idempotency, and risk;
6. executes through application services, not repositories directly;
7. emits an audit record with request, workflow, execution, approval, and AI provenance.

There are deliberately no tools for PLC/SCADA control, permit approval, isolation verification, barrier bypass, safety override, motor/valve operation, or equipment shutdown.

### 6.4 Demonstration and learning boundary

Maintenance adapters emit semantic SDK observation events:

```json
{
  "schema_version": "1",
  "id": "evt-123",
  "session_id": "demo-session-1",
  "principal": {
    "tenant_id": "org-1",
    "actor_id": "worker-123",
    "roles": ["senior_technician"]
  },
  "timestamp": "2026-07-26T02:10:00Z",
  "source": "api",
  "trust": "application_event",
  "application": "skawld-maintenance",
  "action": "measurement_inspected",
  "entity": {"type": "asset", "id": "P-302"},
  "input": {
    "measurement_type": "vibration_velocity",
    "value": "8.1",
    "unit": "mm/s",
    "component_id": "bearing-de"
  },
  "context": {
    "incident_id": "inc-829",
    "source": "manual_entry",
    "verification": "technician_verified"
  }
}
```

Raw taps, mouse coordinates, and screen recordings are not the canonical learning data. A user action may trigger a semantic event only after the application knows its domain meaning.

The SDK analyzer finds repetition, ordering, variation, ambiguity, and corrections. A maintenance extractor may propose a generic SDK workflow version, but maintenance supplies:

- domain tool catalog and schemas;
- approved measurement/unit vocabulary;
- evidence authority and provenance;
- asset/document/workflow applicability;
- prerequisite types such as permit, isolation, gas test, competency, and supervisor;
- risk classification and publish authority.

Compilation creates a `CANDIDATE`; reviewers add applicability, validity, effective/review dates, and approvals before publication. A correction creates an `ImprovementCandidate`, never an in-place workflow mutation.

### 6.5 SDK versioning

- During co-development, a workspace-level `go.work` outside both repositories uses their local module paths. It is developer convenience and is not committed to either repository unless the parent workspace is itself managed.
- CI and release builds ignore local replacements and resolve a tagged SDK dependency from `go.mod`.
- While SDK is `v0.x`, maintenance pins an exact tag (not a branch or pseudo-version for releases), reads release notes, and runs its contract suite before upgrade.
- Breaking SDK changes use a new minor version during `v0` and include migration notes. Patch releases are expected to be compatible bug/security fixes.
- Maintenance may temporarily use a local `go.work` during coordinated changes, but SDK and product commits remain independently reviewable.
- A stable SDK should move to `v1` only after real maintenance use validates the public contracts.

## 7. Domain Model

Business objects are aggregates/value objects, not one struct per table.

### 7.1 Core aggregates

**Organization**

- Tenant and authority boundary.
- Has sites and memberships.
- Exists from day one even for single-company deployments.

**Site**

- Operational/location scope within an organization.
- Holds timezone, policies, and external identifiers.

**ApprovalAuthority**

- Separate from role membership and ordinary permissions.
- Grants a subject or delegated group authority to approve a defined subject/action within organization/site, asset or asset class, workflow, competency, and maximum risk scope.
- Has `valid_from`, `valid_until`, optional delegation provenance, and revocation state.
- Approval evaluation requires both the ordinary permission to review and a current matching authority grant.

**Asset**

- Stable identity, tag, name, class, manufacturer/model, lifecycle status, source-of-truth declaration, external reference when projected, criticality, attributes, and hierarchy links.
- `AssetRelationship` uses typed relationships (`contains`, `part_of`, `drives`, `fed_by`, `protects`, customer extension) rather than a hard-coded seven-level schema.
- Components may be assets when independently identifiable; `AssetComponent` is used for lighter subordinate structure.
- Criticality is human-approved and includes rating plus safety, production, environment, financial, and redundancy dimensions.

**Incident**

- A reported abnormal condition or maintenance need.
- Contains summary, severity, state, asset/site, reporter, occurrence/detection times, symptoms, links to observations/measurements, source reference, and resolution.
- It does not own every execution record.

**MaintenanceExecution**

- The actual unit of work, covering inspection/troubleshooting/repair/verification without requiring a full CMMS work order.
- Links an incident or normal-work purpose, workflow version when used, assigned participants, prerequisites, steps, observations, measurements, decisions, actions, outcome, and approval/report state.
- `WorkReference` links to an externally authoritative work order or provides a lightweight Skawld-owned local draft/reference for small customers. The two modes cannot be silently converted.

**KnowledgeDocument**

- Metadata and validity for OEM manual, SOP, work instruction, drawing, datasheet, inspection/safety procedure, bulletin, or report.
- Owns immutable revisions, applicability, approval status, effective/expiry dates, supersession, ingestion state, and object references.

**MaintenanceWorkflow**

- Product metadata around the SDK workflow identity/version: maintenance purpose, applicability, effective/review dates, lifecycle status, authority, required competencies, and prerequisites.
- Executable structure and evidence live through SDK workflow contracts; applicability remains maintenance-owned.
- Maintenance lifecycle is `DRAFT`, `CANDIDATE`, `VALIDATED`, `PUBLISHED`, `REVIEW_REQUIRED`, or `RETIRED`; map it explicitly to the smaller SDK executable status without changing SDK semantics.
- Applicability evaluation returns `VALIDATED`, `LIKELY_APPLICABLE`, `NOT_VALIDATED`, or `NOT_APPLICABLE`. Only a valid published version can guide automatically; `LIKELY_APPLICABLE` requires an authorized human to accept the use context.

**DemonstrationRecord**

- Product link between a work context and SDK demonstration ID, with consent/capture mode, maintenance taxonomy version, review state, and outcome.

**ShiftHandover**

- Draft/accepted record for a site and shift window: out-of-service equipment, abnormal conditions, open work, isolations/permit references, temporary changes, safety concerns, follow-up, provenance, and acknowledgement.
- AI can prepare a draft only.

### 7.2 Value objects

```text
TenantScope             OrganizationID + optional SiteID
AssetPath               ordered typed hierarchy references
ExternalReference       system + entity type + external ID + version
SourceOfTruth           OWNED_BY_SKAWLD or EXTERNAL_REFERENCE
Criticality             rating + impact dimensions + approval
EngineeringMeasurement  decimal value + unit + type + original value/unit
MeasurementContext      time + source + instrument + quality + verification
EvidenceRef              kind + ID + revision/version + locator + authority
Recommendation           content + evidence + assumptions + unknowns
RecommendationOutcome    recommended/insufficient-evidence + confidence + risk
Applicability            org/site/class/maker/model/service/condition/capability
Prerequisite             type + required + status + verified by/at/source
Provenance               origin type + author/source + timestamp + revision
RiskClassification       informational/advisory/low-risk/safety-significant/critical
SyncVersion              server revision + client mutation ID
```

An externally sourced projection also records source system, external ID, external version/ETag when available, last synchronized time, mapping version, and local fields allowed to be enriched. Synchronization never makes Skawld authoritative by accident.

Engineering values use decimal representations and a controlled unit catalog. Conversion is deterministic, tested, and records the original representation. The LLM may extract a candidate measurement from text but cannot convert or verify it.

### 7.3 Future-compatible, not V1-complete

- Reliability: `FailureMode`, `FailureCause`, `FailureMechanism`, `FailureConsequence`, `Downtime`, `RepeatFailure`, `MaintenanceStrategy`.
- Process safety: `SafetyBarrier`, `BarrierStatus`, `SafetyCriticalElement`, `Impairment`, `OverrideReference`.
- Change: `ChangeEvent` that marks affected documents/workflows `REVIEW_REQUIRED`.
- Workforce: contractor organization, competency, certification, site access.
- Materials: asset BOM and part requirement/availability references sourced from ERP/EAM.

These begin as external references or controlled metadata until a real workflow needs richer behavior.

Domain profiles extend vocabulary, applicability rules, document types, and integration mappings; they do not fork the core runtime:

- **Ports:** cranes (STS/QC, RTG, RMG), reach stackers, terminal tractors, reefer infrastructure, mooring/fender, gates, and electrical distribution.
- **Oil and gas:** rotating/pressure equipment, pipelines, offshore/well maintenance, corrosion, asset integrity, barrier/permit/isolation references.
- **Petrochemical:** turnaround, MOC, pressure/piping/cryogenic equipment, QA/QC, reinstatement, startup/shutdown readiness.
- **Power:** turbine/generator/transformer/switchgear/boiler and outage/availability concepts.
- **General industry:** manufacturing, steel, cement, food/cold storage, warehouse, calibration, safety inspection, and contractor workflows.

Profiles are enabled by configuration and mappings after a real customer validates terminology. They cannot weaken common safety, evidence, authority, tenancy, or workflow-version rules.

### 7.4 Domain events

Initial domain events are typed in-process facts such as:

```text
IncidentCreated
ExecutionStarted
MeasurementRecorded
ObservationRecorded
DecisionRecorded
MaintenanceCompleted
ReportSubmitted
HandoverAccepted
```

They are collected by aggregates and dispatched after commit to in-process handlers. Work requiring retry is inserted transactionally into the PostgreSQL job queue/outbox. There is no external event broker. Introduce external messaging only when a real integration needs durable independent consumers, and keep domain events distinct from integration-event contracts.

## 8. Data Architecture

### 8.0 Source-of-truth boundary

Every authoritative-looking record is classified explicitly:

| Information | Default source of truth |
|---|---|
| Workflow, demonstration, correction, recommendation, evidence trace, SDK execution trace | `OWNED_BY_SKAWLD` |
| Asset and incident | External projection in enterprise mode; `OWNED_BY_SKAWLD` only in configured lightweight mode |
| Work order, inventory, purchasing, PM schedule, labor plan, cost, spare availability, permit/isolation status | `EXTERNAL_REFERENCE` |
| Raw telemetry and operational/control state | Historian/OT system; `EXTERNAL_REFERENCE` |
| Maintenance execution/report | Configured per integration contract; ownership and write-back state are explicit |

Projection records retain `external_system`, `external_id`, external version, synchronization status/time, and allowed local enrichment. Adapters translate source data but do not conceal ownership. Write-back is a separate authorized integration command with idempotency and audit.

| Knowledge/data category | System of record | Search/index | Authority treatment |
|---|---|---|---|
| Asset/site/hierarchy | PostgreSQL or external EAM reference | Relational/FTS | Authoritative only if Skawld is configured owner |
| Incidents/executions/history | PostgreSQL or synchronized external refs | Relational + FTS + selected embeddings | Verified operational records outrank AI summaries |
| Measurements (manual/low frequency) | PostgreSQL | B-tree/time/asset indexes | Original unit/value retained; verification explicit |
| High-frequency telemetry | Existing historian; future adapter | Deterministic aggregates only in Skawld | Raw samples never sent directly to LLM |
| Documents/revisions | Metadata in PostgreSQL; binary in S3 | Chunks + FTS + pgvector | Approved/current/applicable revisions preferred |
| Photos/audio/attachments | S3; metadata and checksums in PostgreSQL | Derived candidates/transcripts in PostgreSQL | Binary is immutable evidence; derivatives versioned |
| Demonstrations | SDK semantic model persisted via PostgreSQL adapter | Relational analysis; selected searchable summaries | Expert source, reviewed separately from official procedure |
| Workflow versions | SDK workflow store via PostgreSQL + maintenance metadata | Applicability/metadata indexes | Only published, effective, applicable versions guide work |
| Corrections | PostgreSQL linked to execution/recommendation/context | Dataset export later | Improvement evidence, never self-modification |
| Recommendations | PostgreSQL append-oriented execution record | Evaluation views | Always includes evidence/unknowns/model/prompt versions |
| Audit | PostgreSQL append-oriented table, later archive/WORM option | Time/actor/entity/action indexes | Separate retention and access from app logs |
| Jobs/outbox/inbox | PostgreSQL | Operational indexes | Infrastructure state, not domain knowledge |

### 8.1 Retrieval policy

Retrieval is two-stage:

1. **Eligibility:** enforce authorization, tenant, site, document approval/currentness, asset/workflow applicability, language, source authority, and date inside the database query boundary. Ineligible items cannot be rescued by a high vector score or removed only after AI sees them.
2. **Independent ranks:** run FTS and cosine vector search over that eligible set.
3. **Fusion:** combine rank positions using RRF, then apply only deterministic domain boosts/penalties. Return locators and score components for inspection.

Do not dump the four knowledge categories into a common vector namespace. Embeddings are secondary indexes over versioned source records. Deleting an embedding never deletes source knowledge.

When HNSW is introduced, benchmark filtered recall against exact search. Use pgvector iterative scans, appropriate partial indexes, or partitioning only when measurements show they improve the actual tenant/site/applicability distribution.

### 8.2 Telemetry boundary

```text
historian / sensor stream
          ↓
read-only integration
          ↓
deterministic validation, unit conversion, filtering, aggregation,
threshold evaluation, trend/anomaly-candidate calculation
          ↓
small contextual evidence object
          ↓
retrieval/reasoning model
```

TimescaleDB is not selected. Evaluate it only when Skawld must retain enough condition-monitoring data that native PostgreSQL layout/partitioning is measured to be inadequate. Prefer querying the customer’s historian.

### 8.3 Standard mappings

- ISO 55001:2024 informs lifecycle, value/risk alignment, authority, and asset-management integration.
- ISO 14224:2016 informs mappable equipment taxonomy and failure/maintenance data vocabulary, particularly for petroleum industries.
- ISA/IEC 62443 informs future zones/conduits, service identity, least privilege, secure update, and OT separation.

Mappings live in documentation and adapters. Similar fields do not mean certification or compliance.

## 9. AI Architecture

### 9.1 Safety envelope

```text
untrusted content / model output
              ↓
JSON schema validation
              ↓
domain validation (IDs, units, applicability, document validity)
              ↓
workflow state and prerequisite validation
              ↓
policy risk decision
              ↓
ordinary permission
              ↓
matching current ApprovalAuthority when approval is required
              ↓
human approval when required
              ↓
idempotent maintenance tool
              ↓
domain mutation + audit
```

LLMs summarize, extract candidates, connect evidence, draft language, and explain uncertainty. Normal code owns calculations, unit conversion, thresholds, permissions, state transitions, tenant boundaries, validity, approvals, and tool execution.

### 9.2 Capability routing

Business code asks for a capability and policy class, not a model name:

```go
type TaskClass string

const (
    TaskExtract   TaskClass = "extract"
    TaskReason    TaskClass = "reason"
    TaskVision    TaskClass = "vision"
    TaskSpeech    TaskClass = "speech"
    TaskEmbedding TaskClass = "embedding"
)

type ModelRequest struct {
    Class           TaskClass
    DataSensitivity string
    RequiredSchema  string
    MaxLatency      time.Duration
    QualityTier     string
}
```

Configuration routes:

- small/cheap model: classification, extraction, short summaries;
- strong reasoning model: ambiguous incident synthesis and workflow candidate extraction;
- vision model: observation candidates;
- speech model: transcription;
- embedding model: retrieval indexes.

The router records provider, model, version/snapshot if exposed, prompt/template version, input/output hashes, latency, tokens/cost where available, and outcome. Model names stay in deployment configuration.

### 9.3 Evidence-first RAG

1. Normalize the question and identify tenant, site, asset, component, task, and time.
2. Retrieve eligible structured facts and approved workflow candidates deterministically.
3. Perform filtered lexical and semantic retrieval over eligible document chunks and similar cases.
4. Build a bounded evidence packet with stable evidence IDs and locators.
5. Request a schema-constrained result.
6. Validate every cited evidence ID against the packet.
7. Apply policy and present the recommendation with evidence, assumptions, unknowns, applicability, confidence, and risk.

Representative output:

```json
{
  "status": "RECOMMENDATION",
  "recommendation": "Inspect lubrication condition before intrusive work.",
  "evidence": [
    {"kind": "workflow", "id": "HV-PUMP", "version": 3, "locator": "step-4"},
    {"kind": "document", "id": "OEM-P302", "revision": "R7", "locator": "8.3"},
    {"kind": "measurement", "id": "m-101", "locator": "vibration_velocity"}
  ],
  "assumptions": ["The reading is RMS velocity at the configured point."],
  "unknowns": ["Energy isolation status", "instrument calibration status"],
  "confidence": 0.78,
  "risk_level": "ADVISORY",
  "requires_human_confirmation": true
}
```

If no adequate eligible evidence exists, status is `INSUFFICIENT_EVIDENCE`; the model cannot substitute uncited general knowledge in a site procedure.

### 9.4 Prompt injection and content isolation

- Documents, OCR, transcripts, EXIF, filenames, and retrieved text are untrusted data, never system instructions.
- Retrieval context uses explicit delimiters and source IDs; provider tools are not exposed during pure summarization/extraction unless required.
- A document cannot grant permission, change policy, select a tenant, or request secret/tool access.
- Tool inputs are reconstructed from validated structured fields and server-side identities.
- Retrieved text is size-limited, MIME-validated, malware-scanned, and logged by hash/provenance.

### 9.5 Workflow learning and correction

Semantic demonstrations are captured through the SDK recorder. `learning.Analyze`/`Compiler` identify supported sequence patterns and produce evidence-linked candidates. Maintenance review adds domain applicability and validity.

Store a human correction as:

```text
recommendation ID and exact version
context snapshot/evidence set
AI proposed action
human selected action
reason (optional but prompted)
actor and authority
workflow/version if involved
subsequent outcome
```

Corrections feed evaluation and later compilation. They do not fine-tune models or modify published workflows automatically.

### 9.6 V1 evaluation instrumentation

Capture enough to compute later:

```text
RetrievalPrecision               judged relevant results / retrieved
EvidenceCoverage                 material claims with valid evidence / claims
RecommendationAcceptance         accepted / shown
HumanOverrideRate                corrected / actionable recommendations
IncorrectNextStepRate            reviewer-marked incorrect / next-step outputs
UnsafeRecommendationRate         safety review failures / recommendations
UnsupportedRecommendationRate    unsupported claims / material claims
WorkflowMatchAccuracy            correct workflow matches / evaluated cases
ReportExtractionAccuracy         correct structured fields / evaluated fields
LLM calls, latency, tokens, cost per incident/execution
```

V1 needs event capture, frozen test fixtures, reviewer labels, and a small CLI/test suite—not a separate evaluation platform.

## 10. Offline Architecture

### 10.1 Local data

Drift stores:

- assigned/current incidents and lightweight asset/context snapshots;
- workflow versions and steps needed for assigned work;
- prerequisites and their last known status;
- measurements, observations, notes, decisions, and checklist state;
- attachment manifests and local encrypted file references;
- pending mutations (outbox), received server changes, sync cursors, and conflicts.

The device does not mirror the tenant database. Data is scoped to signed-in worker, assigned site/work, retention policy, and device capability. Sensitive data is encrypted at rest using platform keystore-protected keys where supported.

### 10.2 Write and sync

```text
user action
    ↓ one local SQLite transaction
local domain row + mutation outbox row
    ↓ UI immediately shows LOCAL_PENDING
connectivity returns / explicit sync
    ↓
POST /sync/push with client_mutation_id and base_server_version
    ↓
server auth + schema + domain + policy + idempotency checks
    ↓
accepted server version OR conflict/rejection
    ↓
GET /sync/pull?cursor=...
    ↓
local transaction applies changes and advances cursor
```

Every mutation has `client_event_id`, idempotency key, device ID, actor, tenant/site, entity type/ID, operation, base server version, `created_at_device`, payload version, attempt count, and sync status. The server records `received_at_server` and the resulting server version. The server inbox guarantees replay returns the original result.

Conflict rules are explicit, not CRDT-based:

- append-only observations/measurements: normally merge by unique ID;
- checklist completion: merge only if step/workflow version matches; otherwise flag review;
- editable text/draft report: optimistic concurrency and human resolution;
- workflow/document/criticality/approval: server-authoritative, never silently overwritten;
- terminal work state: server validates the state machine and prerequisites;
- deleted/retired records: retain local draft and present rejection/reconciliation.

### 10.3 Attachments

1. Create attachment manifest locally with checksum, size, MIME candidate, capture time, and linked entity.
2. Sync the structured record independently.
3. Request a constrained signed upload URL. Multipart is added only if the configured capability and file-size requirement justify it.
4. Upload resumably with checksum and bounded retry.
5. Server places object in quarantine, validates MIME/magic bytes, strips unsafe metadata where appropriate, scans, and marks `AVAILABLE` or `REJECTED`.

A failed photo/audio upload never removes an inspection, measurement, or note. UI shows `LOCAL_ONLY`, `UPLOADING`, `QUARANTINED`, `AVAILABLE`, or `FAILED`.

### 10.4 Hazardous areas

Offline mobile is an option, not an assumption that phones are permitted. The same execution can be captured through an approved intrinsically safe device, outside-area delayed entry, post-job voice entry, or a workstation. Software records capture mode and time; device certification remains outside product scope.

## 11. Security & Industrial Safety

### 11.1 Threat controls

| Threat | Architectural control |
|---|---|
| Prompt injection | Treat retrieved content as data; no policy/tool authority; evidence ID validation |
| Malicious documents | Quarantine, allowlisted MIME/magic bytes, size/page limits, sandboxed extraction, malware scan, no macros/scripts |
| Image metadata | Ignore/strip active or unnecessary metadata; never interpret EXIF text as instruction |
| Unsafe upload/path traversal | Generated object keys, never user paths; normalized display names; checksums and limits |
| Command injection | No shell tool in product agent; typed adapters; subprocess allowlists only in isolated workers |
| SQL injection | Parameterized `pgx`/`sqlc` queries; no model-generated SQL execution |
| SSRF | No arbitrary URL fetch; allowlisted connectors; egress controls; block private/link-local metadata endpoints |
| Secret leakage | External secret injection, redaction, least-privilege service identities, never place secrets in prompts/logs |
| Cross-tenant access | Tenant scope derived from principal, repository guardrails, composite constraints/tests; never accept scope from AI |
| Authorization bypass | Permission checks in application service and tool adapter, not UI/handler only |
| Untrusted model output | Schema/domain/state/policy/permission/approval pipeline |
| Sensitive data exposure | Data classification, scoped retrieval, encryption in transit/at rest, retention, export audit |
| Supply chain | Pinned dependencies/images, scanning, SBOM, signed release artifacts at pilot stage |

### 11.2 Recommendation/action classes

| Class | Examples | V1 execution |
|---|---|---|
| Informational | Search history, summarize approved SOP | May run with normal authorization |
| Advisory | Suggest inspection, explain similar case | Human decides and confirms |
| Operational low risk | Draft work reference/report, notification | Draft or explicit approved action |
| Safety significant | Change maintenance/readiness state, alter operational workflow | Strong authority and human approval; mostly out of MVP |
| Critical | Shutdown, interlock override, motor/valve/setpoint control | **Not representable as executable Skawld tools** |

Automation levels map to those controls:

```text
L0 Observe                 capture and audit
L1 Retrieve Knowledge      eligible evidence retrieval
L2 Recommend               evidence-backed advisory output
L3 Assist                  human-directed checklist, draft, and recording
L4 Execute approved        explicitly approved, idempotent low-risk actions only
```

The MVP emphasizes L0–L3. L4 is limited to administrative low-risk tools such as creating a draft record or notification. Safety-significant and critical industrial execution is outside this model.

### 11.3 Prerequisites and process safety

Workflows may declare references such as `requires_work_permit`, `requires_energy_isolation`, `requires_gas_test`, `requires_supervisor`, and `requires_competency`. Skawld does not certify these conditions itself. A step can proceed only when an authorized person or integrated authoritative system has supplied a current verification. Otherwise it is `BLOCKED`, with an auditable reason.

Safety barriers, impairments, bypasses, MOC, PTW/LOTO, and readiness may be displayed or referenced later. Skawld cannot authorize them. Any change affecting knowledge can mark related workflow/document metadata `REVIEW_REQUIRED`.

### 11.4 OT boundary

- No public-cloud application connection directly into control networks.
- A future edge gateway is deployed in a customer-reviewed zone/conduit architecture, uses certificate service identity, allowlisted read-only protocols, outbound store-and-forward, and signed updates.
- No generic SDK/LLM tool receives an OPC UA/MQTT/vendor protocol client.
- No claim of IEC 62443 compliance is made.

### 11.5 Audit

Operational logs describe software behavior (`HTTP 500`, database timeout, worker retry). Audit events describe accountable product behavior (`technician recorded 8.1 mm/s`, `user viewed SOP R7`, `AI recommended step X`, `supervisor rejected it`, `workflow v3 published`).

Audit events are append-oriented and access-controlled. Corrections are new events, not rewrites. Database privileges prevent the runtime role from updating/deleting audit rows. Pilot readiness adds integrity chaining or external immutable export only if retention/threat requirements justify it.

## 12. Repository Structure

Backend, web, and mobile belong in this product monorepo initially. They share one product release, REST contract, fixtures, design language, and pilot backlog. Splitting them would add cross-repository coordination without independent teams. `skawld-sdk-go` remains separate because it has a generic lifecycle and must be usable by other products.

```text
skawld-maintenance/
├── cmd/
│   ├── api/                    # process composition only
│   ├── worker/                 # async process composition only
│   └── eval/                   # local/offline evaluation CLI (later)
├── internal/
│   ├── asset/
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapter/
│   ├── incident/
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapter/
│   ├── execution/              # inspection, observations, actions, report
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapter/
│   ├── knowledge/              # documents, chunks, retrieval, provenance
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapter/
│   ├── workflow/               # maintenance metadata/applicability/review
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapter/
│   ├── handover/
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapter/
│   ├── reliability/            # later bounded context; no V1 suite
│   ├── identity/               # principal, RBAC, approval authority
│   ├── integration/            # EAM/CMMS/document/edge contracts
│   ├── skawld/                 # only SDK-importing product boundary
│   │   ├── tools/              # SDK Tool implementations
│   │   ├── observers/          # domain event -> SDK observation
│   │   ├── workflows/          # SDK workflow adapters
│   │   ├── policies/           # product risk -> SDK policy
│   │   ├── stores/             # PostgreSQL SDK stores
│   │   ├── extractors/         # SDK learning boundary
│   │   ├── routing/            # model capability routing
│   │   ├── schemas/            # versioned structured outputs
│   │   └── prompts/            # versioned templates
│   └── platform/
│       ├── database/
│       ├── jobs/
│       ├── objectstore/
│       ├── auth/
│       ├── observability/
│       └── clock/
├── api/
│   ├── openapi.yaml
│   └── generated/              # generated API clients/types only
├── migrations/
├── web/
│   ├── src/
│   ├── public/
│   ├── package.json
│   └── vite.config.ts
├── mobile/
│   ├── lib/
│   │   ├── core/
│   │   ├── data/
│   │   ├── sync/
│   │   └── features/
│   ├── test/
│   └── pubspec.yaml
├── deployments/
│   ├── containers/
│   └── compose/
├── docs/
│   ├── adr/
│   ├── standards-mapping/
│   ├── threat-model/
│   └── runbooks/
├── test/
│   ├── contract/
│   ├── fixtures/
│   └── evaldata/
├── scripts/
├── .github/workflows/
├── compose.yaml
├── go.mod
├── Spec.md
├── Plan.md
├── ChangeLogs.md
├── ReadMe.md
└── Agent.md
```

Rules:

- Domain packages do not import HTTP, SQL, S3, Flutter/web, provider SDKs, or concrete clock/ID generators.
- Domain and ordinary application packages do not import `skawld-sdk-go`; only `internal/skawld` may do so.
- Application packages define use cases and transaction boundaries. Define interfaces only at a boundary that has or clearly needs substitution.
- Adapters may import domain/application packages; never the reverse.
- Modules may not query another module’s tables directly. In the monolith they call an application query/port; reporting read models may be an explicit exception.
- `internal/skawld/tools` call application services, never database adapters.
- Do not create `common`, `utils`, or `models` dumping grounds.

## 13. Local Development Setup

Target: Fedora/Linux, native fast loop, containerized stateful dependencies.

```text
native:
  Go toolchain
  API/worker processes
  Node/pnpm + Vite
  Flutter/Android toolchain

Podman Compose or Docker Compose:
  PostgreSQL with pgvector
  local OIDC provider
  optional S3-compatible test target
  optional OpenTelemetry collector/profile
```

Recommended flow:

1. Keep sibling repositories under a common developer workspace, for example `workspace/skawld-sdk-go` and `workspace/skawld-maintenance`.
2. At the workspace root, run `go work init ./skawld-sdk-go ./skawld-maintenance`. Do not use a committed `replace` directive for releases.
3. Start only required dependencies with `podman compose up -d postgres oidc objectstore`.
4. Run `go run ./cmd/api` and `go run ./cmd/worker` natively.
5. Run web through `pnpm dev`; run Flutter on an emulator or approved Android device.
6. Use migrations explicitly and seed only fictional/demo industrial data.
7. Make `make check` or an equivalent task the single local quality entry point.

Compose is useful for reproducibility and pilot packaging; Kubernetes is not required. Support Docker-compatible Compose syntax even when Fedora developers use Podman. Secrets are supplied through ignored local environment files or a developer secret tool, never committed.

Container releases use minimal non-root images, read-only root filesystem where practical, health/readiness endpoints, graceful shutdown, pinned base image digests at release time, and no build toolchain in runtime layers.

## 14. MVP Scope

The proposed MVP list is too broad for one developer if “copilot,” voice, vision, RAG, offline mobile, workflow learning, and polished administration all arrive together. The first sellable proof should prioritize lower-risk knowledge and capture.

### In: product MVP

1. OIDC sign-in, organization/site-scoped RBAC, and separate approval-authority grants.
2. Minimal asset context/hierarchy, imported or locally created; not an asset-management suite.
3. Incident intake and status sufficient for the demo.
4. Maintenance execution with checklist, deterministic measurements, observations, actions, outcome, and report approval.
5. Photo and voice-note attachment capture; transcription may be asynchronous and must be verified.
6. Approved/revisioned document upload, extraction, chunks, and applicability metadata.
7. Eligible-set FTS + cosine retrieval with explicit RRF for approved documents and similar resolved incidents.
8. Evidence-backed copilot recommendation with explicit uncertainty and no autonomous action.
9. AI-generated maintenance report draft with technician/supervisor approval.
10. Semantic demonstration capture from execution events.
11. Minimal workflow candidate review using the SDK, initially with curated demo evidence.
12. Offline mobile slice for an assigned execution, measurements, notes, checklist, and attachments.
13. Append-oriented business/AI/approval audit.
14. Shift handover draft/acceptance as the second enterprise demo, using existing records rather than a broad operations module.

### Sequence cut

The first internal slice can use responsive web for supervisor and a deliberately narrow Flutter field flow. Full technician management, analytics, general work-order planning, broad knowledge administration, and vision analysis are deferred. Voice upload belongs in MVP; automatic structured voice extraction can ship after the core evidence path is trusted.

### Out

```text
CMMS/EAM replacement                 inventory/spares ownership
preventive maintenance scheduler     workforce/contractor management suite
reliability analytics suite          predictive maintenance ML
high-frequency telemetry storage     historian
SCADA/DCS/PLC integration            equipment control
authoritative PTW/LOTO/MOC            process-safety approval
multi-agent system                   fine-tuning/RL/self-modifying workflows
complex workflow DSL                 broad analytics platform
microservices/Kubernetes/Kafka       database-per-tenant
full offline dataset                 CRDT collaboration
```

### Pilot ordering

Present **Demo B, Shift Handover, first** to an enterprise industrial customer. It demonstrates normal-work capture, knowledge retention, open-work/safety context, and administrative relief with lower diagnosis and adoption risk. Then present **Demo A, High Vibration Pump**, which shows the deeper maintenance intelligence loop. For smaller contractors, reverse the order because the pump incident is a more immediate product story.

## 15. First Vertical Demo

### Demo A — High Vibration Pump Incident

Fictional data:

```text
Asset: P-302, Industrial Pump
Asset path: Site / Process Area / Pump System / P-302
Criticality: B (human-approved)
Incident: High vibration
Vibration velocity: 8.1 mm/s, RMS, verified manual reading
Bearing temperature: 94 °C, verified manual reading
```

Preloaded evidence:

- current approved OEM manual revision with cited section;
- current site inspection procedure;
- one resolved similar incident with verified cause/action/outcome;
- one published workflow version with applicability to the manufacturer/model or marked `LIKELY_APPLICABLE` and explicitly accepted by a senior;
- maintenance history and last baseline measurement;
- an obsolete procedure retained but excluded by retrieval policy.

Demo flow:

1. Supervisor creates/imports the incident and assigns an execution.
2. Technician syncs the assignment to Flutter and loses connectivity.
3. Technician records the two measurements, an observation, photo, and voice note locally.
4. The app shows applicable workflow steps and blocks intrusive steps because isolation verification is absent.
5. Connectivity returns; structured events sync before the attachment finishes.
6. Deterministic retrieval selects eligible evidence and similar incidents.
7. Copilot recommends the next non-intrusive inspection, cites workflow/document/history/measurement evidence, lists unknowns, and asks for human confirmation.
8. Technician selects a different next step and records a reason. Skawld stores the correction without changing the published workflow.
9. Technician records findings/action and before/after measurement.
10. AI prepares a structured report draft. Technician edits; supervisor approves.
11. The execution and semantic demonstration become future learning evidence.
12. Reviewer sees an improvement candidate/diff, not an automatically modified workflow.

Acceptance:

- Every evidence link opens the exact current source/revision/locator.
- The obsolete document never influences guidance.
- Unit conversion and threshold displays are deterministic and tested.
- Missing prerequisite visibly blocks the intrusive step.
- Offline/retry does not duplicate measurements or lose attachment metadata.
- The recommendation can return insufficient evidence.
- The report retains AI provenance, edits, and approval.
- No model can execute equipment or safety actions.

### Demo B — Shift Handover

Preloaded context:

- P-302 execution in progress;
- one equipment-out-of-service record;
- a permit/isolation external reference (status displayed, not owned);
- an abnormal condition, pending inspection, temporary modification reference, and safety concern.

Flow:

1. Outgoing supervisor opens the shift window.
2. Skawld gathers authoritative open records and drafts categorized handover content with evidence links.
3. The supervisor corrects priority/wording, adds an unknown, and accepts the handover.
4. Incoming supervisor reviews and acknowledges.
5. Changes and omissions are audited; corrections become knowledge/evaluation signals.

This proves learning from normal work without asking AI to diagnose or authorize operations.

## 16. Implementation Phases

The implementation plan, including Goal, Deliverables, Dependencies, and Acceptance Criteria for Phases 0–5, is maintained in [Plan.md](./Plan.md#implementation-phases). The sequence is normative:

```text
Phase 0  Foundation
Phase 1  Core Maintenance
Phase 2  Copilot
Phase 3  Demonstration Capture
Phase 4  Workflow Learning
Phase 5  Pilot Readiness
```

No phase begins by adding infrastructure. Each phase begins with a vertical behavior and adds adapters only when required.

## 17. First Database Schema

Conceptual schema; exact columns and indexes belong in migrations after this architecture is approved. UUID/ULID policy should be consistent. Tenant-owned tables carry `organization_id`; site-owned tables also carry `site_id`. Mutable aggregates have `version`, `created_at`, and `updated_at`. Human-facing numbers are separate from internal IDs. Every projection-capable table carries `source_of_truth`; `EXTERNAL_REFERENCE` records also carry external system/ID/version and synchronization state.

### Identity and scope

| Table | Purpose / important fields |
|---|---|
| `organizations` | id, name, status |
| `sites` | id, organization_id, code, name, timezone, status |
| `principals` | id, external_subject, display_name, status |
| `memberships` | organization/site scope, principal_id, role |
| `approval_authorities` | subject, organization/site, scope kind/ID, competency, maximum risk, delegation source, valid_from/until, revoked_at |

### Asset context

| Table | Purpose / important fields |
|---|---|
| `assets` | tenant/site, tag, name, class, maker/model, lifecycle status, source_of_truth, external system/ID/version, sync state/time, attributes JSONB |
| `asset_relationships` | parent/child asset IDs, typed relation, validity dates |
| `asset_components` | asset_id, code, name, type, attributes JSONB |
| `asset_criticalities` | asset_id, rating, impact dimensions, rationale, approved_by/at |

### Maintenance work

| Table | Purpose / important fields |
|---|---|
| `incidents` | number, asset, summary, severity, state, detected/occurred/resolved times, source_of_truth, external system/ID/version, resolution summary |
| `maintenance_executions` | incident optional, purpose, asset, workflow ID/version optional, state, assigned/started/completed times, outcome |
| `execution_participants` | execution, principal/worker ref, role |
| `execution_steps` | execution, stable workflow step ref or local step, sequence, state, timestamps, blocked reason |
| `prerequisite_verifications` | execution/step, prerequisite type, status, external ref, verified_by/at, expiry |
| `observations` | execution, asset/component, controlled property/status, narrative, source, verification, observed_at |
| `measurements` | execution, asset/component, type, decimal value, unit, original value/unit, source, quality, instrument ref, verification, observed_at |
| `decisions` | execution/step, question/context snapshot, selected option/action, reason, actor, decided_at |
| `maintenance_actions` | execution/step, controlled action type, narrative, component, outcome, performed_by/at |
| `work_references` | execution, source_of_truth, source system, external ID/version, sync/write-back state, URL, local draft status/summary |
| `maintenance_reports` | execution, revision, structured JSONB, rendered object key, state, generated_by metadata, submitted/approved_by/at |

Observations and measurements are append-oriented; corrections create superseding records or verification events rather than silent history rewrite.

### Knowledge and retrieval

| Table | Purpose / important fields |
|---|---|
| `documents` | type, title, authority/source, owner, applicability summary |
| `document_revisions` | document, revision, approval status, effective/expiry dates, superseded_by, object key/checksum, ingestion state |
| `document_applicability` | revision to org/site/asset/class/maker/model/service constraints |
| `document_chunks` | revision, ordinal, locator, content, content hash, token estimate, FTS vector |
| `embeddings` | source kind/id/revision, provider/model/version, dimensions, metric, vector, input hash, status |
| `retrieval_runs` | query/context hashes, filters, ranked evidence IDs/scores, model/version, latency |

Embedding rows are polymorphic secondary indexes by controlled source kind, not domain entities. If dimension changes, new rows coexist while a new index is built; active model configuration chooses the generation.

### Workflow, learning, AI, and audit

| Table | Purpose / important fields |
|---|---|
| `maintenance_workflows` | SDK workflow ID, purpose, owner, status |
| `maintenance_workflow_versions` | workflow ID/version, effective/review dates, approval, supersession, SDK version reference |
| `workflow_applicability` | version + typed constraints, validation status, approved_by/at |
| `demonstration_links` | SDK demonstration ID, execution/context, capture taxonomy version, review state |
| `recommendations` | context/evidence snapshot, structured output, status, risk, provider/model/prompt version, reviewer outcome |
| `human_corrections` | recommendation/workflow/step, proposed, selected, reason, actor, outcome link |
| `improvement_candidates` | workflow/version target, evidence links, state, reviewer decision |
| `approvals` | subject kind/ID, action, risk, requested/decided principal, status/reason/timestamps |
| `audit_events` | tenant/site, actor, action, entity, before/after or patch JSONB, reason, AI/execution/workflow/approval/request IDs, occurred_at |
| `sync_inbox` | device_id, client_event_id, idempotency key, request hash, created_at_device, received_at_server, result/server version/status |
| `sync_changes` | ordered server change feed per tenant/site |
| `attachments` | linked entity, object key, checksum, MIME/size, capture metadata, upload/scan/availability state |

SDK workflow/demonstration/approval/audit persistence may use SDK-shaped tables or these product tables through adapters. Do not duplicate the same authority in both. This choice is finalized during Task 8 after mapping actual SDK store contracts.

### Jobs

River owns its queue tables. Job payloads contain stable IDs, not full sensitive documents. Domain changes and job insertion occur in one database transaction. API and worker use different PostgreSQL pools with an explicit total connection budget. Each queue/job type has configured concurrency, deadline, attempts, retry classification, idempotency, and observability.

## 18. Initial REST API

Conventions:

- Base `/api/v1`; JSON; UTC timestamps with site timezone metadata where relevant.
- `Idempotency-Key` on create/command/sync endpoints.
- ETags or explicit `version` for optimistic concurrency.
- Cursor pagination; RFC 9457-style problem details.
- Tenant/site scope comes from authenticated membership and path/resource validation, never arbitrary headers from the model.
- Commands are explicit where lifecycle meaning matters.

### Session/context

```text
GET    /auth/login
GET    /auth/callback
POST   /auth/logout
GET    /api/v1/me
GET    /api/v1/sites
GET    /api/v1/approval-authorities?subject_id=&site_id=
```

The web auth endpoints establish/clear the server-side session cookie; they never return a long-lived browser token for `localStorage`. Mobile uses the IdP authorization-code + PKCE flow.

### Assets and incidents

```text
GET    /api/v1/assets?site_id=&query=
POST   /api/v1/assets                         # small-customer mode only
GET    /api/v1/assets/{asset_id}
GET    /api/v1/assets/{asset_id}/history

GET    /api/v1/incidents?site_id=&state=&asset_id=
POST   /api/v1/incidents
GET    /api/v1/incidents/{incident_id}
PATCH  /api/v1/incidents/{incident_id}        # constrained editable fields
POST   /api/v1/incidents/{incident_id}/resolve
```

### Executions

```text
POST   /api/v1/incidents/{incident_id}/executions
GET    /api/v1/executions/{execution_id}
POST   /api/v1/executions/{execution_id}/start
POST   /api/v1/executions/{execution_id}/observations
POST   /api/v1/executions/{execution_id}/measurements
POST   /api/v1/executions/{execution_id}/decisions
POST   /api/v1/executions/{execution_id}/actions
POST   /api/v1/executions/{execution_id}/steps/{step_id}/complete
POST   /api/v1/executions/{execution_id}/complete
POST   /api/v1/executions/{execution_id}/report-drafts
POST   /api/v1/reports/{report_id}/submit
POST   /api/v1/reports/{report_id}/approve
```

### Attachments/documents

```text
POST   /api/v1/attachments/initiate
POST   /api/v1/attachments/{attachment_id}/complete
GET    /api/v1/attachments/{attachment_id}

POST   /api/v1/documents
POST   /api/v1/documents/{document_id}/revisions
GET    /api/v1/documents/{document_id}
POST   /api/v1/document-revisions/{revision_id}/approve
POST   /api/v1/document-revisions/{revision_id}/retire
```

### Search/copilot/workflows

```text
POST   /api/v1/search
POST   /api/v1/incidents/{incident_id}/recommendations
POST   /api/v1/recommendations/{recommendation_id}/feedback

GET    /api/v1/workflows?asset_id=&status=
GET    /api/v1/workflows/{workflow_id}/versions/{version}
POST   /api/v1/workflow-candidates/{candidate_id}/review
POST   /api/v1/workflow-candidates/{candidate_id}/publish
```

### Demonstration/handover/sync

```text
POST   /api/v1/demonstrations
POST   /api/v1/demonstrations/{id}/events     # application-issued semantic types only
POST   /api/v1/demonstrations/{id}/complete

POST   /api/v1/handovers
GET    /api/v1/handovers/{id}
POST   /api/v1/handovers/{id}/prepare-draft
POST   /api/v1/handovers/{id}/accept
POST   /api/v1/handovers/{id}/acknowledge

POST   /api/v1/sync/push
GET    /api/v1/sync/pull?cursor=
```

Avoid a generic `/chat` endpoint. Copilot requests are contextual, typed application use cases.

## 19. Skawld SDK Integration API

The following is an intended shape, not committed code. The first interfaces are product-owned and contain no SDK type. All subsequent SDK implementation/composition code lives inside `internal/skawld`; actual SDK types are used exactly as released and do not cross back into maintenance modules.

```go
// Product application boundary.
type MaintenanceQueries interface {
    AssetContext(ctx context.Context, principal identity.Principal, id asset.ID) (asset.Context, error)
    History(ctx context.Context, principal identity.Principal, id asset.ID) ([]execution.HistoryItem, error)
    SearchEvidence(ctx context.Context, principal identity.Principal, q knowledge.Query) ([]knowledge.Evidence, error)
}

type MaintenanceCommands interface {
    RecordObservation(ctx context.Context, principal identity.Principal, cmd execution.RecordObservation) (execution.Observation, error)
    RecordMeasurement(ctx context.Context, principal identity.Principal, cmd execution.RecordMeasurement) (execution.Measurement, error)
    PrepareReport(ctx context.Context, principal identity.Principal, cmd execution.PrepareReport) (execution.Report, error)
}
```

Maintenance tools implement the SDK’s actual `core.Tool` contract:

```go
type RecordMeasurementTool struct {
    Commands MaintenanceCommands
    Principal PrincipalResolver // server-side execution context -> principal
}

func (t RecordMeasurementTool) Name() string { return "maintenance.record_measurement" }
func (t RecordMeasurementTool) Scope() core.ToolScope { return core.ToolScopeWrite }
func (t RecordMeasurementTool) ParallelSafe() bool { return false }
func (t RecordMeasurementTool) ToolDescriptor() core.ToolDescriptor {
    return core.ToolDescriptor{
        Risk:        core.RiskMedium,
        SideEffect:  core.SideEffectIdempotent,
        Idempotency: core.IdempotencyRequired,
        Permissions: []string{"measurement:record"},
    }
}

// Validate parses a versioned, closed JSON schema into a typed command.
// Execute resolves the principal from trusted context, invokes application
// authorization/domain validation, and returns a structured result.
// ExecuteIdempotent binds the write to the SDK-provided idempotency key.
```

Composition:

```go
registry := tools.NewRegistry()
registry.Register(GetAssetContextTool{Queries: queries, Principal: principals})
registry.Register(SearchApprovedKnowledgeTool{Queries: queries, Principal: principals})
registry.Register(RecordObservationTool{Commands: commands, Principal: principals})

runner := workflow.RegistryRunner{Registry: registry}
executor, err := workflow.NewExecutor(workflow.ExecutorOptions{
    Tools:         runner,
    Policy:        maintenanceRiskPolicy,
    Approvals:     postgresApprovalStore,
    Audit:         postgresAuditSink,
    // Other required SDK options are supplied according to the pinned version.
})
```

Demonstration capture:

```go
demo, err := recorder.Start(ctx, "high_vibration_pump", sdkPrincipal, initialContext)
event := observation.Event{
    // A normalized maintenance semantic event with explicit trust/provenance.
}
event, err = recorder.Capture(ctx, demo.ID, event)
demo, err = recorder.Complete(ctx, demo.ID, result)
```

Candidate compilation:

```go
compiler := learning.Compiler{
    Extractor: maintenanceWorkflowExtractor,
    Tools:     maintenanceToolCatalog,
    Store:     postgresWorkflowStore,
}
result, err := compiler.CompileMultiple(
    core.WithPrincipal(ctx, sdkPrincipal),
    workflowID,
    name,
    demonstrations,
    learning.MultiDemoOptions{},
)
// Save as candidate. Maintenance review adds applicability and validity.
// Publication is an explicit authorized human command.
```

Integration rules:

- Do not expose the SDK’s coding tools (`Bash`, `Write`, etc.) to the maintenance agent.
- Do not use the agent session as the source of business state.
- Store conversation/execution transcripts according to retention and sensitivity policy.
- Translate SDK generic risk into the maintenance action classification without weakening either.
- Evaluate ordinary action permission separately from a matching current `ApprovalAuthority`; an SDK approval record is not itself proof that the actor had domain approval authority.
- Preserve `SourceOfTruth` and external versions when tools read projections or prepare write-back drafts.
- Wrap SDK upgrades behind product contract tests for tool validation, approval pause/resume, idempotency, audit, demonstration recording, and candidate publication.

## 20. First 20 Engineering Tasks

The exact tasks in dependency order, each with Goal, Why now, Files/modules, Implementation, Acceptance Criteria, and Tests, are maintained in [Plan.md](./Plan.md#first-20-engineering-tasks). Tasks 1–20 are normative for initialization; unplanned product infrastructure requires an ADR and owner approval.

## Final Architecture Question: Are We Accidentally Building Another CMMS?

**Not with this scope, but the risk is real.**

The warning signs would be building comprehensive work-order planning, preventive schedules, labor rosters, inventory/procurement, contractor administration, cost accounting, permits, or an authoritative enterprise asset register. Remove or integrate those capabilities instead of expanding them.

Keep only the lightweight operational records required to:

```text
observe work
capture expert knowledge and normal-work context
understand decisions and outcomes
retrieve evidence and operational knowledge
compile reviewable reusable workflows
assist technicians
learn from corrections
preserve organizational memory
```

For enterprise installations, Skawld references or synchronizes CMMS/EAM work and writes approved results back through adapters. For small customers, native incident/execution/work-reference records are deliberately shallow. If a feature does not strengthen the intelligence/knowledge loop or enable a pilot without an existing CMMS, it is outside the product core.

Enforcement is structural, not rhetorical: each projection declares `EXTERNAL_REFERENCE`, retains its external identity/version, rejects unauthorized native mutation, and uses a separate audited write-back command. Skawld-owned workflows, demonstrations, corrections, evidence traces, and recommendations remain `OWNED_BY_SKAWLD`.

## References

These are architectural references, not certifications:

- ISO, [ISO 55001:2024 — Asset management systems requirements](https://www.iso.org/standard/83054.html).
- ISO, [ISO 14224:2016 — Reliability and maintenance data for equipment](https://www.iso.org/standard/64076.html).
- IEC, [IEC 62443-2-1:2024 — IACS asset-owner security program requirements](https://webstore.iec.ch/en/publication/62883).
- PostgreSQL, [Full Text Search documentation](https://www.postgresql.org/docs/current/textsearch.html).
- pgvector, [official repository and capabilities](https://github.com/pgvector/pgvector).
- River, [Go/PostgreSQL job system](https://github.com/riverqueue/river).
- OpenID Foundation, [How OpenID Connect works](https://openid.net/foundation/how-connect-works/).
- Ceph, [Object Gateway and its documented S3-compatible operation set](https://docs.ceph.com/en/latest/radosgw/).
- MinIO, [archived community repository and licensing/distribution notice](https://github.com/minio/minio).
