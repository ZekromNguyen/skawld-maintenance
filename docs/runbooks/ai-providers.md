# AI Provider and Transcription Deployment

The API and worker select their AI providers from environment variables at
startup. The default `deterministic` providers need no credentials and are
used by CI, the seed, and demos; a real deployment switches one or both
provider selections and runs the secret-gated contract tests below.

## Provider selection

Both processes read `config.AI` (validated in
`internal/platform/config/config.go`); an unknown provider value or missing
credentials fail startup.

| Variable | Values | Default |
|---|---|---|
| `STRUCTURED_PROVIDER` | `deterministic` \| `openai` \| `anthropic` | `deterministic` |
| `EMBEDDING_PROVIDER` | `deterministic` \| `openai` | `deterministic` |

### Structured provider credentials

- `openai` (OpenAI-compatible chat completions with JSON output):
  `AI_ENDPOINT` (absolute HTTP(S) URL), `AI_API_KEY`, `AI_MODEL`,
  `AI_MODEL_VERSION`. The adapter posts a chat completion with
  `response_format: {type: json_object}` and expects the model to return the
  exact JSON shape requested in the prompt.
- `anthropic` (Anthropic Messages API): `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL`,
  `ANTHROPIC_MODEL_VERSION`. The endpoint is fixed to the public Messages API.

### Embedding provider credentials

- `openai` (OpenAI-compatible embeddings): `EMBEDDING_ENDPOINT` (absolute
  HTTP(S) URL), `AI_API_KEY` (shared key), `EMBEDDING_MODEL`,
  `EMBEDDING_MODEL_VERSION`. The adapter posts `{input, model}` and records
  the vector dimension from the first response.

## Embedding dimension warning

Switching `EMBEDDING_PROVIDER` after documents are already embedded mixes
vector dimensions inside the single `embeddings` table. A switch must be
paired with re-ingestion:

```sql
-- requires the skawld_owner role or an equivalent
DELETE FROM embeddings;
```

then requeue the document revisions so the ingest worker reprocesses them:
upload a new attachment to each revision (attaching an attachment resets
`ingestion_state` to `QUEUED`), or reset the state directly, e.g.

```sql
UPDATE document_revisions
SET ingestion_state = 'QUEUED', ingestion_error = NULL
WHERE organization_id = '<org-uuid>';
```

Revisions already in `ingestion_state = 'READY'` are claimed but not
re-embedded by the worker, so `DELETE FROM embeddings` alone is not enough.
Do not mix providers in one database.

## Running the contract tests

The secret-gated contract tests perform one real call per adapter and skip
when their credentials are absent:

```bash
export AI_ENDPOINT=https://... AI_API_KEY=... AI_MODEL=... AI_MODEL_VERSION=...
export EMBEDDING_ENDPOINT=... EMBEDDING_MODEL=... EMBEDDING_MODEL_VERSION=...
# and/or
export ANTHROPIC_API_KEY=... ANTHROPIC_MODEL=... ANTHROPIC_MODEL_VERSION=...
go test ./internal/skawld/ -run Contract -v
```

CI runs the same command in the `provider-contracts` job; the job is green
without credentials and executes the contracts once a repository has them
configured as secrets.

## Transcription deployment

The transcription provider is optional and independent of the structured/
embedding selection. Without `TRANSCRIPTION_ENDPOINT`, the API returns 503
for transcription requests (`UnavailableTranscriptionProvider`). To deploy:

| Variable | Purpose |
|---|---|
| `TRANSCRIPTION_ENDPOINT` | absolute HTTP(S) URL of the speech-to-text endpoint |
| `TRANSCRIPTION_API_KEY` | bearer key, optional |
| `TRANSCRIPTION_PROVIDER` | provider name recorded in provenance |
| `TRANSCRIPTION_MODEL` / `TRANSCRIPTION_MODEL_VERSION` | model identity |

The worker posts multipart audio (`file`, `model`,
`response_format=json`) with an `X-Skawld-Source-MIME` header. Every
transcript is stored as an **unverified candidate**; a human must verify or
reject it via `POST /api/v1/transcriptions/{id}/verify` before it becomes
evidence. The transcription contract test
(`TestTranscriptionProviderContract`) exercises the endpoint with a
synthetic silent WAV.
