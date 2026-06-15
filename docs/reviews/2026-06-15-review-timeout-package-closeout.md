# Review timeout and package closeout status

Date: 2026-06-15

## Summary

This change tightens two operator-facing gaps found during long Codex App
orchestration runs:

- external reviewer timeouts were too easy to treat as a vague command failure;
- cleaned package rows did not clearly tell the orchestrator whether a feature
  package looked locally closed or still needed package-level closeout judgment.

## Changes

- Added `review run --timeout-seconds` for short local smoke tests and fixtures.
- Kept `--timeout-minutes` as the normal reviewer-run timeout.
- Records reviewer timeouts as `blocked` reviewer timeout reports and ledger
  routine runs instead of silently skipping the failed reviewer state.
- Added `closeoutStatus`, `closeoutReason`, and `closeoutNextAction` to package
  status rows.
- Status HTML and Markdown now show package closeout hints, including
  `candidate-closed` when all recorded package workers are terminal and no local
  review, cleanup, or blocker pressure remains.

## Evidence Labels

- `local/static`: timeout classification, package closeout hints, status page
  rendering, and tests are local helper evidence only.
- `proxy/advisory`: external reviewer output remains advisory evidence.
- `blocked`: a reviewer timeout records blocked review evidence for that
  reviewer run.
- `direct`: none. This change does not produce runtime, device, provider,
  production, payment, deploy, or live proof.

## Boundaries

- A `candidate-closed` package is not an automatic product-complete claim. It
  only means the local ledger has no same-package review/cleanup/blocker
  pressure.
- External reviewer timeouts do not authorize merge, push, cleanup, release,
  deploy, or proof promotion.
- The orchestrator still decides whether to rerun, import another reviewer,
  record an optional-skipped waiver, or close/defer the package.

## Gates

- `go test ./...`

