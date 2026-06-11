# Budget Policy Report/Eval Local Spec

Task: `TF-CODEX-ORCH-V4-BUDGET-POLICY-REPORT-EVAL-LOCAL`

## Scope

- Continue from the review-only budget policy design note.
- Add a bounded local/static contract for a future budget-policy report or eval
  surface.
- Keep the slice review-only and visibility-only: no scheduler, priority
  engine, automatic killing, automatic dispatch enforcement, merge/push/delete
  automation, or live worker control.

## Local/Static Evidence

- `docs/roadmap.md` was inspected for the current v2.5 budget-policy direction.
- `docs/reviews/2026-06-11-budget-policy-review-only-design.md` was inspected
  as the previous design slice.
- `routines/budget-policy-report.json` now defines the future report contract.
- `examples/routine-reports/budget-policy-report.review-only.json` gives future
  implementation work a concrete RoutineRunReport fixture.
- `docs/routines/README.md`, `docs/routines/harness-map.md`, and
  `docs/v2-usage.md` now document that this is a contract/fixture only, not a
  runnable budget enforcement command.

## Report Contract

The future report must keep these categories separate:

- budget metadata coverage from routine specs and optional ledger tasks;
- local/static pressure warnings from recorded heartbeat or observe output;
- unknown timing states when live runtime or human review elapsed time is not
  recorded;
- recommendations that require Codex App orchestrator or human review.

The report must not:

- create, dispatch, pause, prioritize, reschedule, stop, or kill workers;
- merge, push, delete branches, or clean worktrees;
- mutate the ledger or heartbeat report as a budget response;
- decide dispatch eligibility;
- convert local/static budget evidence into direct runtime/session proof.

## Gates

Local gate results for this slice:

- `go test ./...` passed.
- `go run ./cmd/codex-orchestrator validate-routines --dir routines` passed.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
  passed with local/static evidence and no findings.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
  passed with local/static evidence and no rule hits.
- `git diff --check` passed.

## Residual Risks

- The contract does not prove live Codex App session runtime or human review
  elapsed time; those remain blocked unless future ledger/App events record
  them.
- A future runner could still overstate budget warnings unless it preserves the
  local/static evidence label and keeps unknown timing in blocked evidence.
- Static eval can catch obvious wording or report-shape drift, but human/App
  judgment is still required before pausing dispatch or changing concurrency.
