# v0.3.3 Release Notes

`v0.3.3` makes `codex-orchestrator` easier to run hands-off and easier to
understand when it is running. It keeps the Codex App-first workflow: Codex App
is still the orchestrator, while the helper provides durable local state,
status snapshots, conservative watchdog evidence, and package-level review
handoff material.

## Highlights

- Added package-aware orchestration visibility:
  - `record-task` and `dispatch record` support `--package-id`;
  - `observe`, `status`, and heartbeat reports include `packageSummary`;
  - roadmap scoring now prefers coherent package lanes over unrelated small
    backlog slices.
- Added human-readable status surfaces:
  - `status --write-html` and `status --write-summary` now start with
    "At a Glance" lines;
  - status output highlights integration cleanliness, current package lane,
    missed heartbeat risk, review/blocker/cleanup pressure, dispatch slots, and
    the first recommended action.
- Added hands-off reliability guardrails:
  - helper heartbeat reports can detect missed local heartbeat intervals;
  - macOS LaunchAgent watchdog scripts can emit local/static missed-wakeup
    notifications;
  - `codex-orchestrator watchdog status --repo .` inspects the installed
    watchdog plist, loaded state, last report, summary, and logs.
- Added package-level external review workflow:
  - `pack review` builds portable local/static review material;
  - `review policy check` recommends when a package should use one or two
    external reviewers;
  - `review run --reviewer pi|claude` and `review import` record advisory
    model review results without treating them as merge authorization.

## Why This Release

Real long-running Codex App orchestration exposed two practical gaps:

1. Users could not quickly see what the orchestrator was doing, especially when
   tasks were split across several worker sessions.
2. Missed heartbeat wakeups and failed worktree setup needed durable,
   local/static evidence instead of relying on chat memory.

`v0.3.3` addresses those gaps without turning the helper into a daemon or an
agent operating system. The helper still does not create Codex sessions, merge,
push, clean worktrees, or prove live runtime behavior.

## New Commands And Outputs

Package status:

```bash
codex-orchestrator record-task --id TASK --package-id PACKAGE --worktree /path/to/wt --branch codex/task
codex-orchestrator observe --json
codex-orchestrator status --write-html .codex-orchestrator/status.html --write-summary .codex-orchestrator/status.md
```

Hands-off watchdog status:

```bash
REPO=/path/to/project ./scripts/install-macos-watchdog.sh
codex-orchestrator watchdog status --repo /path/to/project
```

Package external review:

```bash
codex-orchestrator pack review --package-id PKG --task-id TASK --output /tmp/review-pack/PKG
codex-orchestrator review policy check --package-id PKG --risk medium --task-count 4 --json
codex-orchestrator review run --package-id PKG --reviewer pi --pack /tmp/review-pack/PKG --write-report /tmp/pi-review.json
```

## Evidence Boundary

The new watchdog and status surfaces are `local/static` evidence. They can show
that a local check ran, did not run, or saw a missed interval. They cannot prove
why Codex App did not wake a thread, keep a sleeping Mac awake, or replace
Codex App heartbeat automation.

External model review is advisory evidence. A Pi, Claude, DeepSeek, or other
review report can help the orchestrator find issues, but it does not by itself
authorize implementation, merge, push, cleanup, release, deployment, external
service calls, or direct runtime proof.

## Verification Before Publishing

Checks used for this release:

- `go test ./...`
- `bash -n scripts/install-macos-watchdog.sh scripts/macos-watchdog-run.sh`
- `codex-orchestrator watchdog status --repo .`
- `go run ./cmd/codex-orchestrator validate-routines --dir routines --json`
- `go run ./cmd/codex-orchestrator policy check --repo .`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `git diff --check`

The local helper was rebuilt and the installed Codex skill was synced before
publishing.

## Suggested Announcement

`codex-orchestrator v0.3.3` improves App-first Loop Engineering visibility:
package-level status, human-readable status snapshots, missed-heartbeat
detection, macOS watchdog status, and optional package-level Pi/Claude review
handoffs. It remains a conservative helper around Codex App, not a daemon or
autonomous release bot.
