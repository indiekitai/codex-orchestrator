# v0.3.0-beta.1 Release Notes Draft

`v0.3.0-beta.1` should be the first release positioned for external users to
try, not just inspect. It should package the current App-first orchestration
skill, durable helper, routine contracts, and read-only routine runners behind a
clear quickstart.

## Highlights

- Codex App-first orchestrator skill for supervised multi-session development.
- Durable local task ledger with `init`, `record-task`, `observe`, `heartbeat`,
  `status`, and `append-event`.
- Routine contract validation through `validate-routines`.
- Read-only routine runners for:
  - `pr-reviewer`
  - `stale-task-rescuer`
  - `ci-fixer`
  - `release-verifier`
  - `docs-drift-checker`
  - `evidence-label-auditor`
  - `roadmap-next-task-suggester`
- Budget visibility in ledger, observe, and heartbeat reports.
- Conservative evidence labels: `direct`, `proxy`, `local`, `blocked`.
- Cross-platform release binary workflow.
- Beta usability guide for first-time users.

## What Changed Since Alpha

- The helper is now a Go CLI suitable for single-binary distribution.
- Ledger observation distinguishes terminal statuses such as `merged`,
  `released`, and `cleaned` from active/pending work.
- Routine runners produce inspectable JSON reports but do not mutate git or
  create sessions.
- Documentation now separates the Codex App orchestrator layer from the local
  helper layer.
- The roadmap explicitly avoids claiming this is a daemon or full agent OS.

## Install

Download the asset for your platform from GitHub Releases, or build locally:

```bash
go build -trimpath -ldflags='-s -w' -o codex-orchestrator ./cmd/codex-orchestrator
```

Install the skill:

```bash
mkdir -p ~/.codex/skills
cp -R . ~/.codex/skills/delegated-session-orchestrator
```

## Try It

```bash
codex-orchestrator init
codex-orchestrator observe --json
codex-orchestrator validate-routines --dir routines
codex-orchestrator run-routine docs-drift-checker --repo . --json
```

Then open Codex App and ask:

```text
Use $delegated-session-orchestrator for this repository.
Use codex-orchestrator observe --json as durable state.
Create isolated worktree sessions for worker tasks.
Review before merge and keep evidence labels honest.
```

## Beta Boundaries

This release does not:

- create Codex App sessions from the CLI,
- run as a background daemon,
- merge or push automatically from the helper,
- clean worktrees automatically,
- replace human engineering review,
- prove production, hardware, payment, or deployed runtime behavior.

## Verification Before Publishing

- `go test ./...`
- `go build -trimpath -ldflags='-s -w' -o /tmp/codex-orchestrator ./cmd/codex-orchestrator`
- `codex-orchestrator validate-routines --dir routines`
- `codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- GitHub release assets uploaded for darwin/linux/windows matrix.

## Suggested Announcement

`codex-orchestrator` beta packages the loop-engineering workflow I have been
using with Codex App: split roadmap work into isolated sessions, track them in a
durable ledger, run heartbeat/routine checks, review before merge, and keep
evidence labels honest.

It is not a daemon or full agent OS yet. It is the first practical layer: a
supervised outer loop for multi-session coding work.
