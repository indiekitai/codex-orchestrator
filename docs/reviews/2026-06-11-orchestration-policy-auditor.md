# Orchestration Policy Auditor

Date: 2026-06-11

Scope: first V4 policy/eval runnable routine for `codex-orchestrator`

## Summary

Added `run-routine orchestration-policy-auditor` as a read-only local/static
checker for orchestration policy regressions observed in real Codex App usage.
It turns repeated workflow mistakes into deterministic named rules instead of
leaving them only as prose.

## Rules

- `OPA001`: dry-run dispatch barrier. Dry-run wording must not allow worker
  dispatch or session creation without explicit user confirmation.
- `OPA002`: no-main-checkout fallback guard. Setup failure must not lead to
  implementation in the orchestrator/main checkout.
- `OPA003`: continuation guard. Completing one child task must not stop the
  broader queue before checking ledger, roadmap, repo truth, or queue state.
- `OPA004`: delegated worker boundary. Real worker prompt blocks must require
  isolated worktree, no subagents/Paseo, self-review, and no merge/push.
- `OPA005`: evidence promotion boundary. Local/proxy/weak evidence must not be
  promoted to direct proof.

## Changed Files

- `cmd/codex-orchestrator/main.go`
- `cmd/codex-orchestrator/main_test.go`
- `routines/orchestration-policy-auditor.json`
- `README.md`
- `README.zh-CN.md`
- `SKILL.md`
- `docs/routines/README.md`
- `docs/roadmap.md`
- `docs/reviews/2026-06-11-orchestration-policy-auditor.md`

## Evidence

- `local`: Go unit tests cover passed, failed, and blocked policy-auditor
  states.
- `local`: `validate-routines` accepts the new JSON routine spec.
- `local`: `docs-drift-checker` sees the new runnable routine in source, spec,
  README, Chinese README, SKILL, routine docs, and roadmap.
- `local`: `evidence-label-auditor` still passes after adding the new routine.
- `local`: `orchestration-policy-auditor` passes against the current repository
  with no rule hits.
- `blocked`: no Codex App transcript parser is implemented yet; this is static
  repo-local policy/eval, not direct App runtime proof.

## Verification

```bash
go test ./...
go run ./cmd/codex-orchestrator validate-routines --dir routines --json
go run ./cmd/codex-orchestrator run-routine orchestration-policy-auditor --repo . --json
go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json
go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json
```

## Self-review

- Diff boundary: scoped to CLI routine, tests, spec, docs, and this review.
- Forbidden actions: no session dispatch, worktree cleanup, release, tag, or
  external service action.
- Evidence honesty: all claims are `local` or `blocked`; no runtime/direct proof
  is claimed.
- Residual risk: rules are deterministic heuristics and intentionally
  conservative. Future work should add transcript-backed fixtures before using
  this as a stronger classifier.
