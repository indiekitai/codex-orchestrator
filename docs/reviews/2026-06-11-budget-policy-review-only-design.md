# Budget Policy Review-Only Design

Task: `TF-CODEX-ORCH-V4-BUDGET-POLICY-REVIEW-ONLY-DESIGN-LOCAL`

## Scope

- Define the next budget-policy direction after the completed local/static
  `budgetSummary` and `budgetPressure` warning slice.
- Keep the next step at docs/spec review level: the helper may report budget
  facts and policy findings, while the Codex App orchestrator and human reviewer
  remain responsible for decisions.
- Preserve the current boundary: no scheduler, prioritizer, process killer,
  dispatch decision, or budget enforcement is introduced by this slice.

## Current Baseline

- `observe`, `status`, and heartbeat reports expose additive budget summaries
  from ledger task metadata and repo-local routine specs.
- `observe` and heartbeat JSON/Markdown expose local/static `budgetPressure`
  warnings for missing budgets, runtime near/exceeded, review near/exceeded, and
  unknown review elapsed time.
- Routine specs currently define `maxRuntimeMinutes` across all local specs.
  `reviewBudgetMinutes` is allowed by the contract but is not yet populated in
  the checked-in specs.

## Evidence Labels

- `local/static`: budget warnings are computed from repo-local routine specs,
  ledger-shaped task records, and recorded timestamps only.
- `local`: this design is based on local docs/spec inspection and local helper
  gates.
- `blocked`: no direct runtime/session timing proof is claimed. Exact Codex App
  session runtime, live worker progress, and human review elapsed time remain
  blocked unless a future ledger or App event records them.

## Review-Only Policy Boundary

The helper may report:

- which tasks or routine specs are missing runtime or review budget metadata;
- whether a recorded task is near or beyond a locally recorded runtime budget;
- whether a review-ready task is near or beyond a locally recorded review
  budget;
- whether review elapsed time is unknown because the ledger lacks a review-ready
  timestamp;
- which routine families have budget metadata coverage gaps;
- a review-only recommendation such as "needs human review before more
  dispatch" or "budget metadata should be added before this routine becomes
  dispatchable."

The Codex App orchestrator or human reviewer may decide:

- whether to dispatch another worker;
- whether to pause, nudge, review, merge, or mark a task blocked;
- whether budget warnings are acceptable for the current project phase;
- whether a future routine should gain or change runtime/review budgets.

The helper remains forbidden from:

- starting, stopping, killing, prioritizing, or rescheduling workers;
- creating, merging, pushing, or cleaning worktrees as a budget response;
- upgrading local/static budget warnings into direct runtime proof;
- treating missing budget metadata as failure without a human-approved policy;
- making dispatch eligibility decisions by itself.

## Proposed Future Task Breakdown

1. Budget policy review report
   - Input: roadmap, routine specs, optional ledger, and current heartbeat
     report.
   - Output: read-only report that separates metadata coverage, local/static
     warning state, unknown timing state, and human-review recommendations.
   - Acceptance: report emits only `local`, `local/static`, or `blocked`
     evidence and states that it does not enforce budgets.

2. Routine budget metadata coverage pass
   - Input: `routines/*.json` and the routine contract docs.
   - Output: proposed `reviewBudgetMinutes` coverage guidance or docs-only
     rationale for leaving a routine without review budget metadata.
   - Acceptance: no scheduling, priority, or dispatch behavior changes.

3. App-orchestrator decision checklist
   - Input: helper report plus human project context.
   - Output: review checklist for when budget pressure should trigger pause,
     same-task nudge, review, or blocked handoff.
   - Acceptance: checklist is advisory and explicitly requires human/App-layer
     judgment.

4. Optional policy/eval follow-up
   - Input: local review notes or fixtures describing budget-evidence misuse.
   - Output: static rules that catch claims such as "budget warning killed the
     worker" or "local timestamp proves live runtime."
   - Acceptance: static findings are review prompts, not semantic convictions.

## Acceptance Gates

- `go test ./...`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `git diff --check`

## Residual Risks

- Static budget warnings can overstate urgency if ledger timestamps are stale or
  incomplete.
- Review elapsed time remains approximate unless future events record the
  review-ready transition.
- A deeper policy report may still require human project context to decide
  whether budget pressure should pause dispatch.
