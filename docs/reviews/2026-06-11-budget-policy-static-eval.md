# Budget Policy Static Eval Follow-Up

Task: `TF-CODEX-ORCH-V4-BUDGET-POLICY-STATIC-EVAL-LOCAL`

## Scope

- Added bounded local/static eval coverage for budget-policy wording drift in
  the orchestration policy auditor.
- Kept the checker review-only: it reports deterministic local/static
  suspicions and does not dispatch, pause, kill, prioritize, merge, push,
  cleanup, mutate the ledger, or enforce budgets.

## Local/Static Evidence

- Added `OPA008` for budget-policy evidence/control boundary drift.
- Added fixtures:
  - `eval/orchestration-policy-auditor/budget-static-evidence-promotion.json`
  - `eval/orchestration-policy-auditor/budget-helper-control-overclaim.json`
  - `eval/orchestration-policy-auditor/budget-review-only-no-hit.json`
- Updated `eval add-failure` policy-rule validation to accept `OPA006`,
  `OPA007`, and `OPA008`.
- Updated roadmap/routine docs to list `OPA001`-`OPA008` and mark the budget
  static-eval follow-up complete after fixture coverage existed.

## Gates

Required gate results for this slice:

- `go test ./...` passed.
- `go run ./cmd/codex-orchestrator policy check --repo . --json` passed with
  `orchestration-policy-auditor status: passed`, 49 repo-local policy inputs
  scanned, no rule hits, and 19 eval fixtures passed.
- `go run ./cmd/codex-orchestrator eval run --repo . --json` passed with 19
  orchestration-policy-auditor fixtures.
- `go run ./cmd/codex-orchestrator run-routine orchestration-policy-auditor --repo . --json`
  passed with 49 repo-local policy inputs scanned and no rule hits.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
  passed with no `ELA001`-`ELA009` rule hits.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
  passed; key docs mention all runnable routines.
- `git diff --check` passed.

## Residual Risks

- `OPA008` is a deterministic text heuristic, not semantic proof. Findings still
  require human/App-layer review before changing dispatch, concurrency, pause,
  kill, merge, push, cleanup, or budget-enforcement behavior.
- This slice does not prove live Codex App session runtime, worker wall-clock
  state, or human review elapsed time. Those remain blocked unless future direct
  runtime APIs or ledger/App events provide them.
