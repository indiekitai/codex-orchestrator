# Restaurant POS Overnight Feedback Hardening

Date: 2026-06-13

## Scope

This change turns the latest restaurant POS overnight orchestration feedback into
codex-orchestrator hardening. The feedback was local/static operational evidence,
not direct proof of Codex App automation delivery or OS wake behavior.

## Changes

- `pack merge-readiness` no longer fails solely because a worker worktree has
  untracked `.codex-orchestrator/` local state files.
- Merge-readiness records state-dir-only files as local/static residual risk so
  the orchestrator can still see the noise without treating it as business dirty
  work.
- The status page now renders terminal `drain` mode more clearly: when there are
  no active, pending, review, blocked, stale, or cleanup items, the dispatch slot
  metric says the queue is stopped rather than highlighting raw spare capacity.
- The human progress summary now says `queue stopped / no dispatch` for terminal
  drain state instead of implying the orchestrator is waiting for another worker.
- The skill now requires package switches to record concrete reasons such as
  `package-closed`, `local-scope-drained`, `blocked`, `owner-gated`, or
  `shared-blocker-removal`.

## Evidence Labels

- `local/static`: code inspection, unit tests, docs drift check, evidence label
  auditor, and policy/eval checks in this repository.
- `blocked`: this change does not explain missed Codex App heartbeat delivery,
  OS sleep, network, or automation-runner behavior.
- `direct`: none.
- `proxy`: none.

## Gates

- `go test ./...`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `git diff --check`

## Residual Risk

- Heartbeat gaps are still only detectable after a helper/status turn runs. An
  OS-level watchdog can help surface missed wakeups, but it remains local/static
  evidence and does not prove why Codex App did not wake a thread.
- Raw `availableSlots` remains present in JSON for machine consumers. Humans
  should follow `dispatchRecommendation` and the first-screen status summary
  instead.
