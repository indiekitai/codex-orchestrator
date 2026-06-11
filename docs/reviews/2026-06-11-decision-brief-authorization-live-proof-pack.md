# Decision Brief / Authorization / Live Proof Pack Update

Date: 2026-06-11

## Scope

Updated `pack consultation` and `pack merge-readiness` to absorb three useful
maintainer-orchestrator disciplines:

- **Owner Decision Brief**: a blocked or decision-gated task should produce a
  concise, decision-ready brief instead of a vague blocker.
- **Authorization Matrix**: review, implementation, merge, push, cleanup,
  release, and owner ask are separate permissions.
- **Live Proof Gate**: local/static packs must name whether direct live proof
  appears required, whether it is recorded, and whether an item-specific waiver
  would be required.

This does not copy Peter Steinberger's GitHub maintainer workflow wholesale.
The change keeps `codex-orchestrator` focused on Codex App-first
feature-package orchestration and local/static review artifacts.

## Changed Report Shape

`pack consultation` now emits:

- `ownerDecisionBrief`
- `authorizationMatrix`
- `liveProofGate`

`pack merge-readiness` now emits:

- `authorizationMatrix`
- `liveProofGate`
- `acceptanceReport`

The new fields are review aids. They do not dispatch, merge, push, cleanup,
release, edit the ledger, edit git state, call the network, or claim direct
runtime/product/device/provider proof.

## Evidence Labels

- `local`: Go unit tests, report-shape assertions, local docs, and static JSON
  schema generation logic.
- `proxy`: None.
- `direct`: None. No live runtime, production, provider, device, payment,
  hardware, or Codex App automation proof was attempted.
- `blocked`: Actual owner decisions, live proof, physical actions, releases,
  deploys, and item-specific waivers remain outside these local/static packs.

## Boundaries

The authorization matrix is intentionally conservative:

- merge requires a separate orchestrator acceptance decision;
- push requires separate closeout authorization after accepted merge;
- cleanup requires a separate closeout/disposition decision;
- release is never authorized by either pack;
- consultation authorizes only sending the structured decision request.

The live proof gate is heuristic and local/static. It detects likely live-proof
requirements from task metadata, paths, gates, blocker text, and evidence
labels, then reports missing direct evidence instead of inventing proof.

## Verification Plan

Expected checks for this change:

```bash
gofmt -w cmd/codex-orchestrator/main.go cmd/codex-orchestrator/main_test.go
go test ./cmd/codex-orchestrator -run 'TestPackConsultation|TestPackMergeReadiness'
go test ./...
go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json
go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json
go run ./cmd/codex-orchestrator policy check --repo .
git diff --check
```

## Residual Risks

- Live proof requirement detection is deterministic text/path heuristics, not a
  semantic proof classifier.
- The acceptance report is a draft produced from local/static pack inputs; the
  actual accept/reject/blocked decision still belongs to the orchestrator or
  human reviewer.
- Project-specific live proof rules still need project docs and worker task
  contracts.

## Self-Review

- Diff reread: Go schema, report generation, tests, README, Chinese README,
  roadmap, and this review doc were reread after editing.
- Allowed paths: changes are limited to `cmd/codex-orchestrator/**`,
  `README.md`, `README.zh-CN.md`, and `docs/**`.
- Forbidden paths: no `.github/**`, `Formula/**`, release workflow, package
  manager distribution, credentials, or unrelated project files changed.
- Docs drift: user-facing README sections and roadmap now describe the new pack
  fields and preserve local/static evidence boundaries.
- Verification gap: no direct runtime/product/device/network proof was
  attempted or claimed.
