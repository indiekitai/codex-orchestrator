# No Homebrew/Package-Manager Route Review

Task ID: `TF-CODEX-ORCH-V4-DOCS-NO-HOMEBREW-PACKAGE-MANAGER-ROUTE`

## Scope

Updated only public docs allowed by the task:

- `README.md`
- `README.zh-CN.md`
- `docs/roadmap.md`
- `docs/distribution-package.md`
- `docs/beta-usability-package.md`
- `docs/reviews/2026-06-11-no-homebrew-package-manager-route.md`

No Go code, scripts, workflows, release assets, tags, package-manager
implementation, or package-manager publishing path was changed.

## Product Route

The docs now state the intended user mental model:

1. Give the GitHub repository to Codex App.
2. Let Codex App read the repository and install the Codex App skill if needed.
3. Let Codex App decide whether the helper binary is useful for durable
   ledger/routine support.
4. Treat source install and GitHub release binaries as optional helper paths,
   not as the primary product route.

Homebrew, npm wrappers, taps, and other package-manager distribution routes are
explicitly out of scope. They are not described as later package work, optional
convenience work, or beta blockers.

## Local Evidence

- `rg -n "Homebrew|brew|npm wrapper|tap|package manager|package-manager" ...`
  was used to inspect remaining wording. Remaining matches are out-of-scope
  boundary statements or this review note.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --json`
  passed with `local` evidence. It found all runnable routines represented in
  the expected docs.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --json`
  passed with `local` evidence. It scanned 22 repo-local evidence-label input
  files and reported no rule hits.

No direct Codex App session-launch proof, release proof, package-manager proof,
or external distribution proof was produced by this docs-only task.

## Residual Risks

- The repository may still contain historical package-manager artifacts outside
  the allowed docs scope. This task did not edit those paths.
- The GitHub release binary helper path remains documented as optional helper
  installation; this is intentional and separate from package-manager
  distribution.
- The routine evidence is local/static. It does not prove production behavior,
  real user setup behavior, or live Codex App bootstrap execution.

## Self-Review

I reread the diff after editing and checked that the docs preserve the Codex
App-first bootstrap flow while removing wording that presents Homebrew/npm/tap
distribution as planned or desirable. The changed files are all inside the
allowed path list, and no forbidden implementation, script, workflow, release,
tag, or package-manager files were touched.
