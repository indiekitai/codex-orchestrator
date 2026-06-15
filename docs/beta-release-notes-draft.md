# v0.3.10 Release Notes

`v0.3.10` is a small reliability and operator-clarity release for real Codex
App orchestration loops. It focuses on two things that showed up in live use:
reviewers timing out, and package lanes looking "done" without a clear closeout
signal.

## Highlights

- `review run` now supports `--timeout-seconds` in addition to
  `--timeout-minutes`.
- External reviewer timeouts are recorded as `blocked` reviewer-timeout reports
  and ledger routine runs instead of being hidden as generic command failures.
- Package status rows now expose:
  - `closeoutStatus`;
  - `closeoutReason`;
  - `closeoutNextAction`.
- Status HTML and Markdown surface package closeout hints. A package can now
  show `candidate-closed` when all recorded workers are terminal and there is
  no visible local review, cleanup, or blocker pressure.
- Skill and guide docs now explain how to treat reviewer timeouts and package
  closeout hints without promoting local/static evidence into direct proof.

## Why This Release

Long-running orchestration often stalls for practical reasons: an external
review command hangs, a package has no active workers but no explicit "done"
decision, or the orchestrator is unsure whether to rerun, waive, or switch
lanes.

`v0.3.10` makes those states visible. A reviewer timeout becomes a ledgered
review setup issue. A cleaned package row now tells the operator whether it is
only locally quiet or ready for an explicit package closeout decision.

## Evidence Boundary

All new helper outputs are `local/static` or `proxy/advisory` evidence:

- timeout classification and package closeout hints are local/static helper
  evidence;
- external reviewer output remains proxy/advisory;
- a reviewer timeout is blocked review evidence for that reviewer run;
- none of this authorizes implementation, merge, push, cleanup, release,
  deploy, or direct runtime/device/provider proof.

## Verification Before Publishing

Checks used for this release:

- `go test ./...`
- `git diff --check`
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- Pi review attempted; it produced no output within 90 seconds and was treated
  as unavailable reviewer evidence, not as a passing review.

## Suggested Announcement

`codex-orchestrator v0.3.10` improves real Codex App loops by making reviewer
timeouts and package closeout state explicit: no silent waiting, no guessing
whether a cleaned package is actually ready to close.
