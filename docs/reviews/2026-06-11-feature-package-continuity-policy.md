# Feature Package Continuity Policy

Date: 2026-06-11

## Scope

Capture a real TastyFuture orchestration failure mode: unattended continuous
runs can become "safe backlog sweepers" that dispatch unrelated tasks only
because they are local, mergeable, and write-set disjoint. That is safe for git,
but poor for product progress and daily reporting.

## Changes

- Added an explicit `SKILL.md` rule for unattended runs: pick one primary
  feature package or product module, then refill worker capacity from that
  package whenever possible.
- Added `OPA009` to the orchestration policy auditor for unrelated safe-backlog
  dispatch that breaks feature-package continuity.
- Added a fixture for the failure pattern.
- Updated README, Chinese README, routine docs, and roadmap references from
  `OPA001`-`OPA008` to `OPA001`-`OPA009`.

## Evidence

- `local`: `go test ./...` passed.
- `local`: `go run ./cmd/codex-orchestrator eval run --repo . --json` passed
  with 27 orchestration policy fixtures.
- `local`: `go run ./cmd/codex-orchestrator policy check --repo . --json`
  passed with no repo-local policy findings.
- `local`: `git diff --check` passed.

## Boundaries

- No Codex App worker was dispatched.
- No TastyFuture business code was changed.
- No package-manager distribution, Homebrew, npm, tap, release, tag, or
  external service work was done.
- This is local/static policy evidence, not runtime proof that future
  orchestrators will always choose coherent packages.

## Self-Review

- Diff stayed within skill, helper policy/eval, docs, and review files.
- The new policy rule is conservative and skips negated/corrective wording to
  avoid flagging documentation that says not to do the bad behavior.
- The fixture proves the failure class is detectable.
