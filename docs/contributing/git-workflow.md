# Git Delivery Workflow

## Branch lanes

```text
feature/<scope>-<description> → developer → staging → production
```

| Branch | Purpose | Direct push |
|---|---|---|
| `feature/*` | One bounded feature or fix, branched from `developer` | Author only; short-lived |
| `developer` | Reviewed integration | Forbidden |
| `staging` | Release candidate and pre-production verification | Forbidden |
| `production` | Approved production release only | Forbidden |

Create a feature branch with a descriptive kebab-case name:

```bash
git switch developer
git pull --ff-only origin developer
git switch -c feature/copilot-evidence-recommendations
```

Open a pull request into `developer`. Promote the exact reviewed commit by pull
request from `developer` to `staging`, then from `staging` to `production`.
Tag releases on `production`; do not rebuild a different commit for release.

## Commit convention

Use Conventional Commits:

```text
<type>(optional-scope): imperative summary
```

Allowed types: `feat`, `fix`, `docs`, `test`, `refactor`, `build`, `ci`,
`perf`, and `chore`.

Examples:

```text
feat(execution): record verified vibration measurements
feat(copilot): add evidence-backed next-step recommendation
fix(sync): preserve attachment retry after a network timeout
test(workflow): cover rejected applicability expansion
docs(pilot): add restore rehearsal checklist
ci(release): require SBOM before staging promotion
```

One commit should implement one coherent, reviewable change. Include migration,
tests, API contract, and UI changes together only when they are inseparable for
that one feature. Split unrelated work into separate commits.

Before committing:

```bash
git diff --check
make check
```

Run the relevant integration, web, and Flutter checks before requesting review.
Do not commit secrets, local `.env` files, build output, or generated package
archives.

## Required GitHub protections

Configure `developer`, `staging`, and `production` with:

- pull request required;
- one approving review minimum;
- required status checks: backend, web, mobile, container, and package jobs as
  applicable;
- resolved review conversations;
- no force pushes or deletions;
- linear history or the team-approved squash/rebase strategy;
- production deployment approval/environment protection.

The repository cannot enforce GitHub branch protection from source files; the
repository owner configures it in GitHub repository settings.
