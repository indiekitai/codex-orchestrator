# v0.3.0-beta.5 Release Notes

`v0.3.0-beta.5` is a visibility and onboarding release. It keeps the same
App-first workflow, but makes the helper's status surface more useful for
humans and future UI/daemon layers: `observe`, `status`, and heartbeat reports
now expose a compact `jobSummary` and a local/static `projectMap` readiness
signal.

## Highlights

- Codex App-first orchestrator skill for supervised multi-session development.
- Durable local task ledger with `init`, `record-task`, `observe`, `heartbeat`,
  `status`, and `append-event`.
- Routine contract validation through `validate-routines`.
- Routine runners for:
  - `pr-reviewer`
  - `stale-task-rescuer`
  - `ci-fixer`
  - `release-verifier`
  - `docs-drift-checker`
  - `evidence-label-auditor`
  - `orchestration-policy-auditor`
  - `roadmap-next-task-suggester`
  - `budget-policy-report`
- Budget visibility in ledger, observe, and heartbeat reports.
- Jobs/status-style `jobSummary` in observe/status/heartbeat JSON and Markdown.
- Project-map readiness detection through `projectMap`, with
  `docs/CODEBASE_MAP.md` as the recommended repository map.
- Conservative evidence labels: `direct`, `proxy`, `local`, `blocked`.
- Cross-platform GitHub prerelease assets for darwin, linux, and windows.
- Shell completion generation for bash, zsh, and fish.
- Beta usability guide for first-time users.

`ci-fixer` executes trusted gate commands already recorded on a ledger task. It
does not edit, stage, commit, merge, push, clean, or update ledger state, but it
should not be run against an untrusted repository or untrusted ledger.

## What Changed Since Beta.4

- `ObserveSummary` now includes `jobSummary`: total task count, per-status
  counts, and compact rows with id, status, signal, branch, pending worktree id,
  latest timestamp, and next action.
- `ObserveSummary` now includes `projectMap`: a local/static check for common
  project-map files such as `docs/CODEBASE_MAP.md`, with a recommended action
  when no map exists.
- `status` text output now prints jobs counts and project-map readiness.
- Heartbeat Markdown summaries now render "Job Summary" and "Project Map"
  sections.
- The repository now includes `docs/CODEBASE_MAP.md` as a concrete project-map
  example for Codex App before broad orchestration.
- README, Chinese README, `SKILL.md`, v2 usage docs, ledger/heartbeat docs, and
  roadmap docs now describe the new status signals.

## Install

The recommended path is to let Codex App install and explain the setup. Open
Codex App in the repository you want to orchestrate and paste:

```text
I want to try codex-orchestrator in this repository.
Read https://github.com/indiekitai/codex-orchestrator and use it as a Codex
App-first orchestration workflow.
Install the Codex App skill from this repository if needed.
If the Go helper CLI is useful, explain what it does and install/build it if
safe.
Start with a dry run and do not push, deploy, delete worktrees, or make
destructive changes unless I explicitly approve.
```

The CLI can still be installed manually from source/tag or release assets, but
it is meant to be a tool the Codex App orchestrator uses, not a prerequisite the
human must learn before trying the workflow. Homebrew is not required for the
beta path.

## Helper Smoke

```bash
codex-orchestrator init
codex-orchestrator observe --json
codex-orchestrator validate-routines --dir routines
codex-orchestrator run-routine docs-drift-checker --repo . --json
```

Use these commands for verification, demos, or advanced debugging after Codex
App has explained why the helper is useful.

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
- `codex-orchestrator policy check --repo . --json`
- GitHub Actions build matrix passed for darwin/linux/windows.
- GitHub prerelease exists with expected darwin/linux/windows assets.
- Release asset download smoke passed for `darwin_arm64`.

## Suggested Announcement

`codex-orchestrator` beta packages the loop-engineering workflow I have been
using with Codex App: split roadmap work into isolated sessions, track them in a
durable ledger, run heartbeat/routine checks, review before merge, and keep
evidence labels honest.

It is not a daemon or full agent OS yet. It is the first practical layer: a
supervised outer loop for multi-session coding work.
