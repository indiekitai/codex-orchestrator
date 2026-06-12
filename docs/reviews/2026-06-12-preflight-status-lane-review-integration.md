# 2026-06-12 preflight/status/lane/review integration

## Scope

This change tightens the existing App-first orchestration loop around four user-facing gaps:

- preflight readiness before leaving a Codex App orchestrator unattended;
- status pages that show what happened recently, not only current raw buckets;
- package lane guardrails so the orchestrator keeps pushing one feature package instead of filling slots with unrelated safe work;
- package-level external review policy signals in `observe` / `status` output.

## Evidence Boundary

All evidence in this change is `local/static`.

It does not prove Codex App heartbeat delivery, OS wake behavior, remote CI, production, pre/prod, device, payment, provider, or hardware state. The helper still does not create Codex sessions, dispatch workers, merge, push, clean worktrees, deploy, or keep a Mac awake.

## Implementation Notes

- Added `codex-orchestrator preflight`.
- `observe` now includes `packageLaneGuard`, `preflight`, and `timeline`.
- `status --json`, `status --html`, and `status --write-summary` expose the same new surfaces.
- Package rows now derive `reviewRequired`, `reviewDecision`, and `reviewNextAction` from the built-in review policy.
- HTML status now shows Preflight, Lane Guard, and Timeline before raw job tables.
- Markdown status now includes Package Lane Guard, Preflight, and Timeline sections.

## Verification

- `go test ./...`
- New tests cover preflight warnings, package review policy signals, lane guard, status JSON, and timeline output.

## Residual Risks

- `preflight` can report missed local heartbeat gaps, but cannot explain whether the root cause was Codex App automation, machine sleep, OS power state, or thread scheduling.
- Package review policy is derived from package id and task count. It is intentionally conservative and should remain advisory until a project-specific policy file says otherwise.
- The static status page is still a file, not a live UI or daemon.
