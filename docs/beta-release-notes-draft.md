# v0.3.0-beta.3 Release Notes Draft

`v0.3.0-beta.3` tightens the public Codex App-first positioning after the
first beta package. It keeps the same helper/routine package, but makes the
entrypoint easier to understand: `codex-orchestrator` is the product, repo,
Codex App skill, and helper CLI name.

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
- Cross-platform GitHub prerelease assets for darwin, linux, and windows.
- Shell completion generation for bash, zsh, and fish.
- Beta usability guide for first-time users.

## What Changed Since Alpha

- The public skill name is now `codex-orchestrator`; users no longer need to
  learn a second skill name.
- README and docs now describe the project as a supervised Codex App outer
  loop, not a complete Loop Engineering runtime or standalone daemon.
- The helper is now a Go CLI suitable for single-binary distribution.
- Ledger observation distinguishes terminal statuses such as `merged`,
  `released`, and `cleaned` from active/pending work.
- Routine runners produce inspectable JSON reports but do not mutate git or
  create sessions.
- Documentation now separates the Codex App orchestrator layer from the local
  helper layer.
- The roadmap explicitly avoids claiming this is a daemon or full agent OS.
- The distribution package now documents release-asset install, source/tag
  install, shell completions, and the App-first boundary that keeps package
  manager installs optional.

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
