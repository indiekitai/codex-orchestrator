# v0.3.4 Release Notes

`v0.3.4` is a usability closeout release for the App-first orchestration loop.
It builds on `v0.3.3` status/preflight work and focuses on three owner-visible
problems: old ledger history making status pages look scattered, new projects
lacking starter planning files, and feature packages needing a clear closeout
checkpoint.

## Highlights

- Added package closeout status:
  - `codex-orchestrator pack status --package-id PKG`;
  - embeds package acceptance and package summary;
  - reports whether a package is ready for orchestrator acceptance, blocked,
    not ready, rejected for fixup, or waiting for external review.
- Added first-project onboarding templates:
  - `codex-orchestrator init --write-templates`;
  - writes non-overwriting local templates for orchestration policy, package
    plan, and project map under `.codex-orchestrator/`.
- Reduced legacy ledger noise:
  - old terminal ungrouped tasks stay in JSON `jobSummary.rows` for audit;
  - status surfaces expose `legacyTerminalUngrouped`, `visibleRows`, and
    `ungroupedNonTerminal`;
  - package lane guard no longer warns when the only ungrouped tasks are old
    cleaned/merged/rejected/abandoned history.
- Refined README / Chinese README / skill docs so Codex App users can discover
  the new flow without learning the helper first.

## Why This Release

Real project orchestration showed that a technically correct ledger can still
feel confusing:

1. Old completed tasks without `packageId` made a fresh status page look like
   the current work was scattered.
2. New projects needed the same project map / package plan / orchestration
   policy shape, but this lived too much in chat.
3. After several related workers landed, the orchestrator needed one compact
   local/static answer to "can this package close?"

`v0.3.4` keeps the same conservative boundary: the helper does not create Codex
sessions, merge, push, clean worktrees, deploy, or prove direct runtime/device
behavior. It gives the Codex App orchestrator better local facts for those
decisions.

## New Commands And Outputs

Initialize starter templates:

```bash
codex-orchestrator init --write-templates
```

Package closeout status:

```bash
codex-orchestrator pack status --package-id CHECKOUT-COUPONS --json
codex-orchestrator pack status \
  --package-id CHECKOUT-COUPONS \
  --write-report .codex-orchestrator/reviews/CHECKOUT-COUPONS-status.json
```

Status rows now distinguish current-action jobs from legacy terminal history:

```json
{
  "legacyTerminalUngrouped": 38,
  "ungroupedNonTerminal": 0,
  "visibleRows": []
}
```

## Evidence Boundary

All new reports in this release are `local/static` evidence. They can help the
Codex App orchestrator decide what to review next, but they do not authorize
merge, push, cleanup, release, deploy, provider actions, or direct live proof by
themselves.

External model review remains proxy/advisory evidence. It can block or inform a
package closeout decision, but the orchestrator still owns the final
accept/reject/block decision.

## Verification Before Publishing

Checks used for this release:

- `go test ./...`
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `go run ./cmd/codex-orchestrator eval run --repo . --json`
- `go run ./cmd/codex-orchestrator status --repo . --json`
- `go run ./cmd/codex-orchestrator preflight --repo . --json`
- `git diff --check`
- Pi read-only proxy/advisory review of the local diff

The local helper was rebuilt and the installed Codex skill was synced before
publishing.

## Suggested Announcement

`codex-orchestrator v0.3.4` makes App-first orchestration easier to read and
close out: package status reports, starter project templates, and legacy ledger
noise control. It still stays conservative: Codex App runs the loop; the helper
provides local ledger/status/review evidence.
