# macOS Watchdog Fallback

Date: 2026-06-12

This change adds an OS-level fallback for hands-off orchestration runs. The
goal is not to replace Codex App heartbeat, but to make missed App wakeups
visible when the Mac is awake and `launchd` can run a local check.

## Change

- Added `scripts/macos-watchdog-run.sh`, a one-shot runner that invokes
  `codex-orchestrator heartbeat --check-only --count 1`, writes watchdog
  heartbeat report and summary files, and sends a macOS notification when the
  report says `heartbeatStatus.status=missed`. `--check-only` keeps the
  external watchdog from appending App heartbeat events and masking missed
  Codex App wakeups.
- Added `scripts/install-macos-watchdog.sh`, which installs a per-project user
  LaunchAgent for the runner.
- Updated `SKILL.md`, English/Chinese README, `docs/v2-usage.md`, roadmap, and
  codebase map with the watchdog usage and boundaries.

## Evidence

- `local`: shell syntax checks cover the new scripts.
- `local`: one-shot watchdog smoke against a temporary git repo produced
  watchdog heartbeat artifacts.
- `local/static`: docs and policy gates keep the watchdog described as an
  external warning layer, not as an orchestrator runtime.
- `blocked`: this does not keep a sleeping Mac awake, create Codex sessions,
  dispatch workers, review, merge, push, cleanup, or prove whether a missed
  wakeup came from Codex App automation, OS power state, or thread scheduling.
