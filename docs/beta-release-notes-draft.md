# v0.3.9 Release Notes

`v0.3.9` adds a small feature-package desired-state layer to the Codex
App-first orchestration workflow. The goal is to keep long runs focused on one
coherent product/module outcome instead of treating several clean worker slices
as a finished feature package by accident.

## Highlights

- Added `pack spec`:
  - generates a package spec template;
  - checks required sections such as outcome, scope, gates, evidence
    boundaries, evaluation matrix, exit condition, blocked condition, and
    waivers.
- Added `pack eval`:
  - builds a local/static evaluation matrix from ledger tasks;
  - checks task commit state, recorded gates, evidence boundaries, and
    package-level integration proof;
  - treats an `active` ledger task with a clean commit after `baseCommit` as
    reviewable by reading git/worktree truth.
- Added `pack reconcile`:
  - compares desired package spec, evaluation matrix, and `pack status`
    closeout;
  - reports drift before the orchestrator claims a feature package is done or
    dispatches the next same-package worker.
- Added starter templates:
  - `.codex-orchestrator/packages/example-package/spec.md`;
  - `.codex-orchestrator/packages/example-package/evaluation.md`.
- Updated the Codex skill and docs to make package specs/evaluation matrices
  part of feature-package planning and closeout.

## Why This Release

Real Codex App orchestration does not fail only because a worker branch is
dirty. It also fails when a run slowly turns into many individually mergeable
slices without a clear package outcome, proof matrix, or exit condition.

`v0.3.9` makes that package-level desired state explicit. It gives the
orchestrator a lightweight way to ask:

- What is this package supposed to accomplish?
- Which proof layers exist?
- Which proof layers are still missing?
- Is package closeout aligned with the original goal?

## New Commands

Generate a package spec:

```bash
codex-orchestrator pack spec \
  --package-id CHECKOUT-PACKAGE \
  --write-template .codex-orchestrator/packages/checkout/spec.md
```

Build an evaluation matrix:

```bash
codex-orchestrator pack eval --package-id CHECKOUT-PACKAGE --json
```

Reconcile desired state and observed closeout:

```bash
codex-orchestrator pack reconcile \
  --package-id CHECKOUT-PACKAGE \
  --spec .codex-orchestrator/packages/checkout/spec.md \
  --json
```

## Evidence Boundary

All new helper outputs are `local/static` planning and review aids. They do not
dispatch workers, merge, push, cleanup, release, deploy, call external
services, or prove direct runtime/device/provider behavior. The Codex App
orchestrator still makes the final accept/reject/block/next-dispatch decision.

## Verification Before Publishing

Checks used for this release:

- `go test ./...`
- `git diff --check`
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- package spec/eval smoke with a temporary git repository
- `/Users/tf/.local/bin/codex-orchestrator --help`

## Suggested Announcement

`codex-orchestrator v0.3.9` adds feature-package desired state: package specs,
evaluation matrices, and reconcile reports so Codex App orchestration can keep
one product/module lane honest instead of mistaking clean slices for a closed
feature package.
