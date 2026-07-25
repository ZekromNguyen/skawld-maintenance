# ADR 0002: SDK Version Gate

Status: Accepted, blocked on upstream release  
Date: 2026-07-26

## Context

The adjacent SDK repository has one public tag, `v0.1.0`. That tag declares the old module path `github.com/skawld/skawld-sdk-go` and does not contain the workflow, observation, learning, policy, approval, or audit packages required by the accepted product architecture.

Those packages currently exist only in an uncommitted SDK working tree whose module path is `github.com/ZekromNguyen/skawld-sdk-go`.

## Decision

- Do not use a `replace` directive in this repository.
- Do not pin Maintenance to an uncommitted SDK working tree or an unusable tag.
- Establish `internal/skawld` with product-owned ports now.
- Add the SDK module dependency and compile/behavioral contract tests immediately after the SDK publishes a clean tag with the correct module path and required packages.
- A developer may use a workspace-level `go.work` for coordinated local verification, but CI and release builds may not depend on it.

## Release gate

Phase 0 cannot be marked fully complete until the SDK release gate passes:

1. clean SDK working tree;
2. correct module path;
3. tagged workflow/observation/learning/policy/audit APIs;
4. Maintenance contract suite green without `replace`.

