# TastyFuture Status / Acceptance Feedback

## Scope

This is a local/static codex-orchestrator follow-up from a real TastyFuture
orchestrator run. It does not include TastyFuture business code, private
artifacts, live device proof, provider proof, or production proof.

## Findings Addressed

- Status snapshots could drift because `status.md` / `status.html` were current
  while an older `.codex-orchestrator/status.json` still showed a stale active
  worker.
- Raw `availableSlots` was still too visually prominent and could be mistaken
  for permission to dispatch unrelated filler work.
- `pack acceptance` could fail after a successful cleanup because the worker
  worktree had already been removed.
- `roadmap score` could rank an old blocker note above current roadmap work
  when the blocker was resolved by a later cleaned task with slightly different
  wording.
- Continuous package closeout could pause too conservatively when one external
  reviewer was unavailable, even though another configured reviewer such as Pi
  was available.

## Changes

- `status` now accepts `--write-report` and writes a sibling `status.json` by
  default whenever `--write-html` or `--write-summary` is used without an
  explicit report path.
- `dispatchRecommendation` now includes `capacityOnly` and `capacityWarning` so
  users and agents can distinguish capacity from dispatch permission.
- `pack acceptance` now supports post-cleanup mode for terminal `merged`,
  `released`, and `cleaned` ledger tasks whose worktree is absent after normal
  closeout. It uses ledger terminal state and recorded gates as local/static
  evidence, while stating that fresh worktree diff proof is unavailable.
- `roadmap score` now applies a conservative blocker-token overlap demotion for
  terminal ledger tasks, reducing stale blocker suggestions from old review
  notes.
- Skill and docs now document status JSON refresh, capacity-only dispatch
  semantics, post-cleanup acceptance, and reviewer-unavailable fallback.

## Evidence Labels

- `local/static`: helper output, ledger state, docs, tests, and deterministic
  policy/eval checks.
- `proxy/advisory`: external reviewer availability or imported reviewer output.
- `blocked`: Codex App heartbeat delivery, OS wake behavior, live production,
  device, payment, provider, and runtime proof remain outside this change.

## Gates

- `go test ./...`: passed.
- `go run ./cmd/codex-orchestrator policy check --write-report /tmp/codex-orchestrator-policy-check.json`: passed.
- `go run ./cmd/codex-orchestrator eval run --write-report /tmp/codex-orchestrator-eval-run.json`: passed.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --write-report /tmp/codex-orchestrator-docs-drift.json`: passed.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --write-report /tmp/codex-orchestrator-evidence-labels.json`: passed.
- `git diff --check`: passed.
