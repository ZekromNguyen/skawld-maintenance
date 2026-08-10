## Summary

<!-- What single feature, fix, or operational change does this PR deliver? -->

## Delivery scope

- [ ] Feature branch is based on `developer`.
- [ ] Commits follow Conventional Commits and are atomic/reviewable.
- [ ] No unrelated refactor or generated build output is included.

## Verification

- [ ] `git diff --check`
- [ ] Relevant Go tests / `make check`
- [ ] Relevant PostgreSQL/S3 integration tests
- [ ] Web lint/test/build, if web changed
- [ ] Flutter analyze/test/native package validation, if mobile changed
- [ ] Migration upgrade/rollback plan reviewed, if schema changed

## Safety and operations

- [ ] No AI output directly executes an industrial-control or safety-critical action.
- [ ] Authorization, tenant/site scope, audit, and source-of-truth boundaries are preserved.
- [ ] Required documentation, runbook, OpenAPI, and evaluation fixtures are updated.
