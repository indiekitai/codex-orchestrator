# v0.3.6 Release Notes

`v0.3.6` is a trust-loop hardening release for the Codex App-first
orchestration workflow. It turns recent research and real project feedback into
local/static controls for developer-agent misalignment: constraint drift,
unsupported completion claims, overreach, setup mistakes, heartbeat gaps, and
package-lane drift.

## Highlights

- Added a local misalignment event log:
  - `codex-orchestrator misalignment record`;
  - `codex-orchestrator misalignment report`.
- Added constraint-stack snapshots on recorded tasks:
  - latest user instruction;
  - active constraints and authorities;
  - evidence boundary;
  - package switch reason.
- Added `claimVerification` to merge-readiness, package acceptance, and
  orchestrator acceptance reports, so completion claims are checked against
  local evidence before acceptance.
- Added a `trustRisk` block to `observe`, `status --write-summary`, and
  `status --html`.
- Added `OPA010`, a deterministic policy/eval rule for tests/completion claims
  made without evidence-bound verification.
- Added `codex-orchestrator version` / `--version` and release-build ldflags so
  published helper binaries can report the tag they were built from.
- Updated English and Chinese docs, the roadmap, and the installed skill
  guidance for the new trust-loop surfaces.

## Why This Release

Long-running coding-agent work does not only fail through bad code. It often
fails through loop misalignment: the agent forgets constraints, overstates
evidence, treats setup failures as pending work, jumps product lanes, or says
something is complete before the repo/worktree/gate evidence supports it.

`v0.3.6` makes those failure modes visible in the local ledger and status
surface. The goal is not automatic trust. The goal is reviewable trust: a
human or orchestrator can see which claims are supported, which are missing
evidence, and which pushback events still need resolution.

## New / Changed Commands

Record a local/static misalignment event:

```bash
codex-orchestrator misalignment record \
  --category self-report-mismatch \
  --task-id TASK \
  --note "claimed tests passed without command evidence"
```

Generate a read-only misalignment insights report:

```bash
codex-orchestrator misalignment report --repo . --json
```

Record a worker contract snapshot:

```bash
codex-orchestrator record-task \
  --id TASK \
  --package-id PACKAGE \
  --worktree /path/to/worktree \
  --branch codex/task \
  --constraint "do not touch payments" \
  --authority "worker may commit only" \
  --evidence-boundary "local/static only"
```

The existing review commands now include `claimVerification`:

```bash
codex-orchestrator pack merge-readiness --task-id TASK --json
codex-orchestrator pack acceptance --package-id PACKAGE --json
```

`policy check` and `eval run` now include `OPA010`.

Check the helper binary version:

```bash
codex-orchestrator version
codex-orchestrator --version
```

## Evidence Boundary

All new outputs in this release are still `local/static` evidence. They do not
prove model intent, production behavior, live device behavior, external
provider state, or Codex App runtime delivery.

`claimVerification` can block or require human review, but it is not automatic
merge authority. Missing evidence should be treated as `blocked` or
`needs-human`, not silently upgraded to direct proof.

## Verification Before Publishing

Checks used for this release:

- `go test ./...`
- `go run ./cmd/codex-orchestrator eval run --json`
- `go run ./cmd/codex-orchestrator policy check --json`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine orchestration-policy-auditor --repo . --json`
- `go run ./cmd/codex-orchestrator help`
- `git diff --check`

The local helper should be rebuilt and the installed Codex skill synced before
publishing.

## Suggested Announcement

`codex-orchestrator v0.3.6` adds a trust-loop layer for Codex App orchestration:
misalignment events, constraint snapshots, evidence-bound claim verification,
trust-risk status, and an OPA010 regression guard for unsupported completion
claims.
