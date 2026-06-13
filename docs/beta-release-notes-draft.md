# v0.3.5 Release Notes

`v0.3.5` is a public-usability and real-run hardening release for the
Codex App-first orchestration workflow. It keeps the same product boundary:
Codex App runs the supervised multi-session loop, while the helper provides
local ledger, status, review, update, and policy evidence.

## Highlights

- Added a local self-update path:
  - `codex-orchestrator self-update`;
  - `codex-orchestrator self-update --from-github`;
  - `codex-orchestrator self-update --with-helper`.
- Split the public README into a short human-facing homepage plus full English
  and Chinese guides.
- Simplified the install/update surface around the App-first workflow:
  - users can paste one bootstrap prompt into Codex App;
  - release binaries remain optional helper assets;
  - Homebrew/npm/tap/package-manager routes stay out of scope.
- Anonymized the public restaurant POS case study and review notes so the repo
  no longer exposes the private project name.
- Hardened status and review behavior from real overnight orchestration:
  - terminal `drain` mode now says the queue is stopped instead of highlighting
    raw spare capacity;
  - `pack merge-readiness` no longer fails solely because a worker has
    untracked `.codex-orchestrator/` local state files;
  - package switches must record concrete reasons instead of following idle
    capacity.

## Why This Release

Real project usage exposed two usability gaps:

1. Public readers saw a README that was useful for agents but too long for
   humans.
2. Long-running orchestration status could confuse capacity with permission:
   `availableSlots` looked dispatchable even when the run was drained, and
   local `.codex-orchestrator/` state files could make merge-readiness look
   failed even when business code was clean.

`v0.3.5` turns those lessons into default behavior and documentation.

## New / Changed Commands

Self-update the installed Codex skill and, when requested, the helper binary:

```bash
codex-orchestrator self-update
codex-orchestrator self-update --from-github
codex-orchestrator self-update --with-helper
```

`self-update` updates the local skill/helper only. It does not dispatch
sessions, mutate project ledgers, merge, push, deploy, or clean worktrees.

`pack merge-readiness` now separates state-dir-only local changes:

```text
.codex-orchestrator/ changes are local orchestration state only
```

That remains local/static evidence and a residual risk, but it no longer blocks
a merge-readiness pack as business dirty work.

## Evidence Boundary

All new helper outputs in this release are still `local/static` evidence. They
can help a Codex App orchestrator decide what to inspect, but they do not
authorize implementation, merge, push, cleanup, release, deploy, provider
actions, external-service mutation, direct runtime proof, or OS wake proof.

External model review remains `proxy/advisory` evidence. It can block or inform
a package closeout decision, but the orchestrator owns the final
accept/reject/block decision.

## Verification Before Publishing

Checks used for this release:

- `go test ./...`
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `go run ./cmd/codex-orchestrator eval run --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `go run ./cmd/codex-orchestrator status --repo . --json`
- `go run ./cmd/codex-orchestrator preflight --repo . --json`
- `go run ./cmd/codex-orchestrator self-update --source . --dry-run --json`
- `git diff --check`

The local helper was rebuilt and the installed Codex skill was synced before
publishing.

## Suggested Announcement

`codex-orchestrator v0.3.5` makes the App-first loop easier to adopt and safer
to leave running: self-update, a shorter README, anonymized public case studies,
clearer drain status, and merge-readiness that ignores local state-dir noise.
