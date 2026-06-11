# v0.3.0-beta.4 Release Notes

`v0.3.0-beta.4` is a small hardening release after external review. It keeps
the same App-first workflow, but fixes naming drift, tightens the `ci-fixer`
safety boundary, makes evidence labels consistently four-bucket
`direct`/`proxy`/`local`/`blocked`, and includes the post-beta.3 release
publishing fix.

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
  - `roadmap-next-task-suggester`
- Budget visibility in ledger, observe, and heartbeat reports.
- Conservative evidence labels: `direct`, `proxy`, `local`, `blocked`.
- Cross-platform GitHub prerelease assets for darwin, linux, and windows.
- Shell completion generation for bash, zsh, and fish.
- Beta usability guide for first-time users.

`ci-fixer` executes trusted gate commands already recorded on a ledger task. It
does not edit, stage, commit, merge, push, clean, or update ledger state, but it
should not be run against an untrusted repository or untrusted ledger.

## What Changed Since Beta.3

- `agents/openai.yaml` now uses the public `codex-orchestrator` skill name.
- `ci-fixer` docs and routine spec now state that it runs trusted recorded
  gates rather than being a fully read-only routine or automatic code fixer.
- `SKILL.md` and `examples/ledger.example.json` consistently use all four
  evidence labels: `direct`, `proxy`, `local`, and `blocked`.
- Ledger JSON writes now use a temporary file plus atomic rename to reduce the
  risk of a truncated durable ledger.
- `scripts/install.sh` now uses the same trimmed build flags as release assets.
- Local gate execution now uses a non-login shell and a Windows `cmd /C`
  fallback when needed.
- The release workflow now runs `go vet ./...` in addition to `go test ./...`.
- README file trees, Chinese terminology, and device-proof language were
  cleaned up for public use.
- The release publishing helper includes the missing-release lookup fix that
  landed on main after `v0.3.0-beta.3`.

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
