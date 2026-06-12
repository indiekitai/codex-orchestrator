# Hands-Off Readiness Guard

Date: 2026-06-12

This change generalizes the TastyFuture night-run lesson: missed wakeups are not
only an overnight risk. They apply whenever a user starts orchestration and then
walks away from the computer.

## Change

- Added `HeartbeatStatus` to heartbeat reports.
- Added `heartbeat --missed-after`; when omitted, the threshold defaults to
  three times the configured interval.
- The helper reads the previous heartbeat event from
  `.codex-orchestrator/events.jsonl` and reports `status=missed` when the gap
  exceeds the threshold.
- Added a hands-off readiness section to `SKILL.md`.
- Updated English and Chinese README guidance to make status snapshots,
  generic heartbeat prompts, missed-run detection, and optional OS watchdogs
  part of the normal workflow.

## Evidence

- `local`: focused Go test covers a five-hour heartbeat gap with 20-minute
  interval and 45-minute missed threshold.
- `local`: helper smoke generated a heartbeat report containing
  `heartbeatStatus.status=missed`.
- `blocked`: this does not prove or fix Codex App automation delivery,
  machine sleep, OS power state, or thread scheduling. It only makes gaps
  visible after the next successful wakeup.
