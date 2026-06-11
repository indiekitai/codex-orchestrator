# Harness Map

This map keeps routine design grounded in feedback-loop engineering. Each
routine needs feedforward guidance, feedback sensors, and control boundaries.

| Harness | Feedforward Guides | Feedback Sensors | Control Boundaries |
|---|---|---|---|
| Git/worktree | Task contract, allowed paths, base commit, branch name | `git status`, `git log`, `git diff`, ledger history | No destructive cleanup without terminal state or human approval |
| CI/local gates | Failing command, CI job URL, expected gate | Command output, exit code, CI rerun result | Do not disable tests or change secrets to pass CI |
| Review | Worker handoff, roadmap/progress docs, self-review | Current diff, docs drift scan, forbidden path scan, narrow tests | Do not merge blocked evidence or unverified direct-proof claims |
| Budget policy | Routine budget metadata, ledger timestamps, heartbeat budget report | Local/static budget summary, pressure warnings, unknown timing flags | Report only; do not schedule, prioritize, enforce, or control workers |
| Runtime proof | Browser/API/device instructions, expected user-visible outcome | Screenshot, API response, device log, database row, app log | Label local/proxy/direct honestly; pause for human-operated hardware |
| Stale rescue | Ledger record, heartbeat report, stale threshold | Worktree existence, branch status, recent commit/diff/thread status | Do not abandon or overwrite useful work silently |

V2.5 only provides the contract and validation layer. V3 can add runnable
adapters for specific harnesses once the output format is stable.
