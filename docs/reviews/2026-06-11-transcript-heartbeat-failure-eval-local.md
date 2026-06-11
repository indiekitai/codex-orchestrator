# Transcript / Heartbeat Failure Eval Local Slice

Date: 2026-06-11

Scope:

- `cmd/codex-orchestrator/main.go`
- `eval/orchestration-policy-auditor/`
- `README.md`
- `README.zh-CN.md`
- `docs/roadmap.md`
- `docs/reviews/2026-06-11-transcript-heartbeat-failure-eval-local.md`

Change:

This slice keeps the roadmap item local/static and extends the existing
`orchestration-policy-auditor` fixture suite instead of introducing private
transcript parsing.

New fixtures:

- `transcript-heartbeat-stale-fixed-task-id.json`: stale heartbeat instruction
  keeps watching a fixed old task id instead of rediscovering repo, ledger,
  worktree, and queue truth. Expected hit: `OPA006`.
- `transcript-pending-worktree-treated-running.json`: a `pendingWorktreeId` is
  counted as a running worker before a real thread, worktree, branch, and setup
  confirmation exist. Expected hit: `OPA007`.
- `transcript-orchestrator-writes-worker-code.json`: after setup failure, the
  orchestrator uses the main checkout to write delegated implementation code.
  Expected hit: `OPA002`.

Existing fixtures already cover:

- child task completed but the orchestrator stops the broader loop:
  `child-complete-without-queue-proof.json`;
- heartbeat bound to the literal `current` placeholder:
  `heartbeat-current-target-binding.json` and
  `heartbeat-current-binding-review-note.json`;
- setup failure fallback to the main checkout:
  `setup-failure-main-fallback.json` and
  `human-review-main-checkout-fallback-transcript.json`;
- local/static/proxy evidence promoted to direct proof:
  `evidence-promotion.json`,
  `human-review-evidence-promotion-transcript.json`, and
  `budget-static-evidence-promotion.json`;
- delegated worker prompts missing core boundaries:
  `worker-boundary-missing.json` and
  `worker-boundary-forbidden-paths-missing.json`.

Narrow helper change:

- `OPA006` now catches action-style stale fixed task id heartbeat wording in
  addition to the literal `current` placeholder.
- `OPA007` now catches action-style `pendingWorktreeId` counted as a running
  worker before setup confirmation, while preserving the existing chat/prompt
  ledger guard.

Evidence labels:

- `local`: all fixture inputs are repo-local deterministic JSON fixtures.
- `local`: transcript wording is sanitized reconstruction, not private
  transcript parsing.
- `blocked`: private transcript validation is not included.
- `blocked`: this does not prove Codex App runtime, scheduler, thread,
  production, pre/prod, device, payment, hardware, or direct orchestration
  behavior.

Run:

```sh
go run ./cmd/codex-orchestrator eval run --repo . --json
go run ./cmd/codex-orchestrator run-routine orchestration-policy-auditor --repo . --json
```

Expected result:

- `eval run` passes the `orchestration-policy-auditor` suite with 22 local/static
  fixtures.
- `orchestration-policy-auditor` scans repo-local policy inputs with no rule
  hits after the docs update.

Residual blocked items:

- No private transcript ingestion or redaction pipeline exists in this slice.
- No Codex App automation/runtime replay is performed.
- Findings remain heuristic local/static suspicions and require human review
  before treating them as confirmed orchestration defects.
