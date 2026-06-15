# v0.3.8 Release Notes

`v0.3.8` is a review-hardening release for the Codex App-first orchestration
workflow. It backports lessons from the related Claude Code orchestration
workflow without changing the core Codex App model: Codex App still owns worker
sessions, and the helper remains a local/static ledger, status, and review aid.

## Highlights

- Added reviewer finding counts to `review import`:
  - `--p0`, `--p1`, `--p2`, `--p3`, and `--other-findings`.
  - Imported external review output remains `proxy/advisory` evidence.
- Added a package-level `findingTracker` to `pack acceptance`.
  - Open P0/P1 findings block package acceptance until fixed or explicitly
    waived.
  - P2 findings stay visible as review debt and should be escalated if they
    survive later package batches.
- Added a package-level `integrationGate` to `pack acceptance`.
  - Multi-task packages now surface missing package-level integration,
    end-to-end, smoke, build, or post-merge gate evidence.
- Added roadmap scorer hints for shared resources:
  - route/navigation registries;
  - localization/copy resources;
  - protocol/API contracts;
  - database schema/migrations;
  - dependency injection or service registries;
  - shared config/feature flags.
- Added roadmap scorer hints for common P1 review patterns:
  - DTO/serialization field drift;
  - state-machine transition coverage;
  - tenant/store scoping filters;
  - integer cents and currency rounding;
  - nullable external/provider fields;
  - idempotency and unique-constraint handling.
- Updated the Codex skill so worker self-review explicitly covers those common
  risk patterns before handoff.

## Why This Release

Real multi-session orchestration is weakest at feature-package boundaries:
small worker branches can each look clean while the package still has open
review findings, shared-resource risk, or no integration proof.

`v0.3.8` makes those risks visible in the local/static review layer. It does
not automate the final engineering decision. It gives the orchestrator better
evidence before deciding whether to accept, reject for fixup, or block a
package.

## New / Changed Commands

Import external reviewer output with finding counts:

```bash
codex-orchestrator review import --package-id PKG --reviewer pi \
  --file /tmp/pi-review.md --status passed --p1 0 --p2 2 --p3 1
```

Generate package acceptance with finding and integration summaries:

```bash
codex-orchestrator pack acceptance --package-id PKG --json
```

Score roadmap candidates with shared-resource and common-risk hints:

```bash
codex-orchestrator roadmap score --repo . --json
```

## Evidence Boundary

All new helper outputs are `local/static` review aids. External reviewer output
is still `proxy/advisory`. These reports do not authorize implementation,
merge, push, cleanup, release, deploy, external-service mutation, or direct
runtime/device/provider proof.

## Verification Before Publishing

Checks used for this release:

- `go test ./...`
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `go run ./cmd/codex-orchestrator validate-routines --dir routines --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine orchestration-policy-auditor --repo . --json`
- `go run ./cmd/codex-orchestrator roadmap score --repo . --json`
- `/Users/tf/.local/bin/codex-orchestrator --help`
- `git diff --check`

## Suggested Announcement

`codex-orchestrator v0.3.8` adds package review hardening: reviewer finding
counts, package acceptance finding tracking, integration-gate reminders, and
roadmap risk hints for shared resources and common P1 failure modes.
