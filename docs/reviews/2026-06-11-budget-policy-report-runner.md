# Budget Policy Report Runner Review

## Scope

Implemented a conservative local/static `run-routine budget-policy-report`
runner for the existing `routines/budget-policy-report.json` contract.

The runner reads only repo-local files:

- `docs/roadmap.md`
- `docs/routines/README.md`
- `routines/*.json`
- optional `.codex-orchestrator/ledger.json` when present
- optional `.codex-orchestrator/heartbeat-report.json` when present

## Evidence Label

Evidence label: `local/static`.

The runner does not emit `direct` or `proxy` evidence. Budget metadata coverage
and heartbeat `budgetPressure` warnings are reported under `local`. Missing
live Codex App session runtime, worker wall-clock state, and human review
elapsed time are reported under `blocked`/unknown.

## Gates

- `go test ./...`: passed.
- `go run ./cmd/codex-orchestrator run-routine budget-policy-report --repo . --json`: passed; emitted local/static report with blocked unknown live timing.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`: passed with top-level README/SKILL coverage restored and `docs/v2-usage.md` included as an additional required routine-reference doc.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`: passed.
- `go run ./cmd/codex-orchestrator policy check --repo . --json`: passed.
- `git diff --check`: passed.

## Boundary Check

Confirmed the runner does not schedule, prioritize, pause, kill, dispatch,
merge, push, delete, clean worktrees, mutate the ledger, mutate heartbeat
reports, or enforce budgets.

The docs-drift checker required docs surface includes `README.md`,
`README.zh-CN.md`, `SKILL.md`, `docs/routines/README.md`, and
`docs/v2-usage.md`; `docs/roadmap.md` remains checked when present. This keeps
the existing top-level docs/SKILL audit coverage while adding the usage doc to
the required routine-reference surface.

## Residual Risks

- The budget report is static visibility only. It cannot prove live runtime or
  human review elapsed time without future direct timing evidence from Codex App
  or another approved runtime source.
- Missing `reviewBudgetMinutes` on existing routine specs is reported as
  advisory local/static evidence, not as a failure or enforcement rule.
- Budget-policy static eval remains a follow-up for detecting future wording
  drift such as promoting local/static timestamp evidence to direct runtime
  proof.
