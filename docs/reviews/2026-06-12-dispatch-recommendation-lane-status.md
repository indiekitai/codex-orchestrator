# 2026-06-12 Dispatch Recommendation Lane Status

## Scope

Converted the latest restaurant POS rewrite usage feedback into helper and skill changes:

- added a first-class `dispatchRecommendation` block so raw `availableSlots`
  is no longer treated as permission to dispatch;
- made active or pending package workers return `recommended=false` with a
  reason and next action;
- split package status into `currentLane`, `historicalReviewDebt`, and cleaned
  package history while keeping the legacy `rows` list;
- updated HTML/Markdown/CLI status output so humans and orchestrators see the
  same dispatch signal;
- updated skill guidance to forbid filling free slots with unrelated "safe"
  worker tasks while a package worker is active or pending.

## Evidence Label

local/static. This is ledger, git worktree, status rendering, and policy text
evidence. It does not prove Codex App heartbeat delivery, OS wake behavior,
production state, device proof, payment/provider behavior, or remote CI.

## Validation

- Focused package/status tests during implementation.
- `go test ./...`
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `go run ./cmd/codex-orchestrator eval run --repo . --json`
- `git diff --check`

## Self-Review

- Changes are scoped to helper/status output, documentation, and the skill
  rules.
- No worker dispatch, merge, cleanup, release, package-manager work, or external
  side effect.
- The new recommendation remains local/static and intentionally does not claim
  live Codex App automation proof.
