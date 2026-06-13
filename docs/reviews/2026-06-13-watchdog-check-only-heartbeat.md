# Watchdog Check-Only Heartbeat

Date: 2026-06-13

## Context

A real project orchestration run showed a monitoring gap where the Codex App
thread did not receive visible heartbeat turns for roughly an hour. Installing
the macOS watchdog made the gap visible, but review found a subtle evidence
problem: the watchdog runner called the normal `heartbeat` command, which also
appended a fresh heartbeat event. That could make future reports look healthy
even when only the external watchdog ran and Codex App did not wake the
orchestrator thread.

## Change

- Added `codex-orchestrator heartbeat --check-only`.
- `--check-only` still runs observe, missed-gap inspection, JSON report output,
  Markdown summary output, and optional JSON stdout.
- `--check-only` does not append a `heartbeat` event to `events.jsonl`.
- Updated `scripts/macos-watchdog-run.sh` to call
  `heartbeat --check-only --count 1`.
- Updated skill and helper docs to separate App heartbeat events from external
  watchdog checks.

## Evidence Labels

- `local`: Go unit test verifies check-only preserves missed heartbeat reporting.
- `local`: Go unit test verifies check-only leaves the events file unchanged.
- `local/static`: macOS watchdog remains a local reporting layer.
- `blocked`: this still does not prove why Codex App missed a wakeup, keep a Mac
  awake, create Codex sessions, review, merge, push, cleanup, or prove live
  project progress.

## Operator Guidance

Use normal heartbeat from a real Codex App monitor turn:

```bash
codex-orchestrator heartbeat --count 1 \
  --write-report .codex-orchestrator/heartbeat-report.json \
  --write-summary .codex-orchestrator/heartbeat-summary.md
```

Use check-only heartbeat from an external watchdog:

```bash
codex-orchestrator heartbeat --check-only --count 1 \
  --write-report .codex-orchestrator/watchdog-heartbeat-report.json \
  --write-summary .codex-orchestrator/watchdog-heartbeat-summary.md
```

