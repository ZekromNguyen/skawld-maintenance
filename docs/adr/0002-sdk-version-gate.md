# ADR 0002: SDK Version Gate

Status: Accepted, gate satisfied
Date: 2026-07-26
Resolved: 2026-07-26 with `skawld-sdk-go` `v0.2.0`

## Context

At the time this ADR was created, `v0.1.0` declared the old module path
`github.com/skawld/skawld-sdk-go` and did not contain the workflow,
observation, learning, policy, approval, or audit packages required by the
accepted product architecture.

The SDK subsequently published `v0.2.0` from commit
`38125435233dd3bd0dd2966d3eb38d49f2639db4` with the canonical module path
`github.com/ZekromNguyen/skawld-sdk-go` and the required public packages.

## Decision

- Do not use a `replace` directive in this repository.
- Pin exactly `github.com/ZekromNguyen/skawld-sdk-go v0.2.0`.
- Keep product-owned ports and all SDK translation in `internal/skawld`.
- Maintain compile-time and behavioral contract tests against the pinned tag.
- A developer may use a workspace-level `go.work` for coordinated local verification, but CI and release builds may not depend on it.

## Verification

The release gate is satisfied:

1. the tagged module resolves through normal Go module verification;
2. `go.mod` contains the exact tag and no `replace`;
3. only `internal/skawld` and the isolated SDK contract-test package import SDK
   packages;
4. maintenance identity maps to SDK tenant/actor identity and canonical
   policy-safe roles;
5. contract tests cover provider streaming, a strict maintenance-shaped tool,
   deterministic execution, idempotency, approval pause/resume,
   requester/approver separation, audit emission, semantic demonstration
   recording, multi-demonstration candidate compilation, exact human review,
   deterministic evaluation gates, and publication;
6. learned candidates remain candidates until the explicit review/evaluation
   publication path succeeds.

## Remaining boundaries

- `ApprovalAuthority` remains product-owned and separate from SDK role
  authorization. A future product approval adapter must satisfy both before
  calling an SDK approval decision.
- Product workflow/applicability persistence over PostgreSQL belongs to later
  workflow phases; this ADR does not authorize Phase 3 implementation.
- Production structured, embedding, speech, and vision provider selection
  remains deployment work and is not implied by pinning the generic SDK.
