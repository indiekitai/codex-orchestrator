# Thread Topology / Pulse-Router-Inbox

Date: 2026-06-14

## Context

A public Codex power-user workflow described durable Pulse, Log, Inbox, and
Router threads. This matched a real gap in `codex-orchestrator`: the project
ledger/status model handled worker lifecycle, but not the surrounding
long-lived thread topology.

## Change

- Added starter templates:
  - `.codex-orchestrator/thread-map.md`
  - `.codex-orchestrator/pulse-threads.md`
- Added `threadMap` to `observe` / `status` outputs.
- Added a preflight `thread-map` warning when no thread map exists.
- Added README, Chinese README, full guide, Chinese full guide, v2 docs, skill,
  roadmap, and research notes for Project Orchestrator / Pulse / Inbox /
  Router / Log roles.
- Added policy/eval rule `OPA011` to catch Router/Inbox/Pulse/Log wording that
  grants execution authority such as implementing code, dispatching workers,
  merging, pushing, deploying, or cleaning worktrees.

## Evidence Labels

- `local`: unit tests cover starter templates and `threadMap` observe/status
  detection.
- `local`: policy fixture covers Router thread execution overreach.
- `local/static`: thread-map files are coordination state only.
- `proxy`: public durable-thread workflow posts and Codex App discussion
  informed the design.
- `blocked`: no direct Codex App runtime proof, thread-to-thread messaging
  proof, or automation-delivery proof is claimed.

## Boundary

This does not turn `codex-orchestrator` into a full agent operating system.
Router, Inbox, Pulse, and Log are coordination roles. The Project Orchestrator
remains the role that owns worker dispatch, review, merge, push, cleanup, and
package closeout.
