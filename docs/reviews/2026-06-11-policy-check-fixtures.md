# Policy Check Fixtures Review

Date: 2026-06-11
Scope: V4 local/static policy check command and orchestration policy eval fixtures

## Result

Added `codex-orchestrator policy check` as the first product-facing V4
policy/eval command. It runs the existing read-only
`orchestration-policy-auditor` and then checks JSON fixtures under
`eval/orchestration-policy-auditor/`.

The initial fixtures cover known orchestration failure classes:

- dry-run dispatch without explicit approval;
- setup failure fallback into the orchestrator/main checkout;
- stopping the larger queue after one child task;
- delegated worker prompts missing mandatory boundaries;
- local/proxy evidence promotion to direct proof.

## Evidence

### Local

- `go test ./...` passed.
- `go vet ./...` passed.
- `go run ./cmd/codex-orchestrator validate-routines --dir routines --json`
  passed.
- `go run ./cmd/codex-orchestrator policy check --repo . --json` passed:
  - `orchestration-policy-auditor status: passed`;
  - current repo scan found no `OPA001`-`OPA005` hits;
  - 6 fixtures from `eval/orchestration-policy-auditor/` passed.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
  passed.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
  passed.
- `go run ./cmd/codex-orchestrator run-routine orchestration-policy-auditor --repo . --json`
  passed.
- `git diff --check` passed.
- `git diff --cached --check` passed.

### Blocked / Not Claimed

- This is local/static policy evidence only.
- No Codex App session was dispatched.
- No daemon, merge automation, runtime worker control, or production proof was
  added.
- `eval add-failure`, `eval run`, and `rules propose` remain future V4 work.

## Self-Review

The change keeps policy/eval read-only and local. It does not mutate ledgers,
git state, worktrees, releases, or Codex sessions. The fixture runner uses the
same `OPA001`-`OPA005` rules as the policy auditor, then compares expected
rule-hit counts so future rule changes have deterministic regression coverage.
