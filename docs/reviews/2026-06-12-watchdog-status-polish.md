# Watchdog Status Polish Review

Date: 2026-06-12

## Scope

- Added a read-only `codex-orchestrator watchdog status` command for the macOS
  external watchdog fallback.
- Exposed local/static watchdog state:
  - expected or discovered LaunchAgent plist;
  - `launchctl print` loaded status when available;
  - last watchdog heartbeat report and summary;
  - launchd stdout/stderr log paths;
  - one-shot runner stdout/error paths;
  - missed heartbeat status from the last report.
- Updated shell completions, README, Chinese README, v2 usage docs, roadmap, and
  the installer success message.

## Evidence Labels

- `local/static`: CLI status inspection, plist/report/log file presence,
  `launchctl` output, and local tests.
- `blocked`: no direct Codex App automation delivery proof is claimed; the
  watchdog can show missed local checks but cannot prove whether the cause was
  Codex App, sleep, power state, or thread scheduling.

## Boundaries

- The command does not install, uninstall, load, unload, dispatch worker
  sessions, merge, push, clean worktrees, or keep the Mac awake.
- The watchdog remains an OS-level warning/reporting layer. Codex App heartbeat
  remains the primary orchestrator wakeup.

## Verification

- `go test ./cmd/codex-orchestrator -run 'Watchdog|Completion' -count=1`
- `bash -n scripts/install-macos-watchdog.sh scripts/macos-watchdog-run.sh`
- `go run ./cmd/codex-orchestrator watchdog status --repo .`
- `go run ./cmd/codex-orchestrator watchdog status --repo . --json`
- `go test ./...`
- `git diff --check`
- `go run ./cmd/codex-orchestrator validate-routines --dir routines --json`
- `go run ./cmd/codex-orchestrator policy check --repo . --write-report /tmp/codex-orchestrator-policy-check.json --json`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --write-report /tmp/codex-orchestrator-docs-drift.json --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --write-report /tmp/codex-orchestrator-evidence-label.json --json`

The current repository smoke reports no installed LaunchAgent and no watchdog
report yet. That is an expected local/static status for this repo, not a
failure of the status command.

## Pi Review

Pi review was run read-only against the package diff. Verdict: no blocking
issues. Actionable suggestions:

- Avoid real `launchctl` calls in tests. Fixed by injecting a test override for
  the loaded-status probe.
- Avoid collision-prone `repo-unknown` suffix when `cksum` is unavailable.
  Fixed by falling back to Go `crc32`.
- Reduce substring risk in plist discovery. Fixed by matching the `REPO` plist
  key's string value instead of any raw path substring.

## Self-Review

- Diff reread: changes are limited to CLI command surface, watchdog scripts
  messaging, docs, tests, and this review record.
- Forbidden paths: no package-manager distribution, Homebrew/npm/tap, session
  dispatch, merge/push/cleanup automation, release tagging, or production paths
  were touched.
- Docs drift: README, Chinese README, v2 usage docs, roadmap, installer output,
  shell completions, and review record were updated.
- Residual risk: `launchctl` status is best-effort local/static evidence; a
  missing or not-loaded LaunchAgent must still be reviewed by the orchestrator
  or human operator before assuming hands-off reliability.
