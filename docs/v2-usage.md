# v2 Usage: Codex App + Go Helper

v2 makes the orchestration loop recoverable. The Codex App orchestrator still
creates worker sessions, reviews branches, merges, pushes, and cleans worktrees.
The Go helper stores durable state and produces heartbeat reports from repo
truth.

The helper is intentionally conservative. It does not create Codex sessions,
merge, push, delete branches, or clean worktrees.

## Install The Helper

Build directly:

```bash
go build -o codex-orchestrator ./cmd/codex-orchestrator
```

Or install to `~/.local/bin`:

```bash
scripts/install.sh
```

If you install elsewhere:

```bash
BIN_DIR=/usr/local/bin scripts/install.sh
```

After a GitHub release is published, users can also download the matching
prebuilt binary from the Releases page:

```bash
tar -xzf codex-orchestrator_darwin_arm64.tar.gz
chmod +x codex-orchestrator_darwin_arm64
mv codex-orchestrator_darwin_arm64 ~/.local/bin/codex-orchestrator
```

## Initialize A Project Ledger

Run this from the repository you want to orchestrate:

```bash
codex-orchestrator init
```

This creates:

```text
.codex-orchestrator/ledger.json
.codex-orchestrator/events.jsonl
```

Keep those files untracked for private local runs. Commit an example/template
only when the team intentionally wants shared state shape.

## Record Dispatch Setup

After the Codex App orchestrator starts worker setup, record the dispatch fact
immediately. If the App returns only a pending worktree setup ID, do not leave
that ID in heartbeat text or chat memory:

```bash
codex-orchestrator dispatch record \
  --task-id API-AUTH-LOCAL \
  --title "Auth endpoint implementation" \
  --package-id AUTH-PACKAGE \
  --thread-id optional-thread-id \
  --branch codex/api-auth \
  --pending-worktree-id pending-worktree-id-from-codex-app \
  --allowed 'src/auth/**' \
  --allowed 'tests/auth/**' \
  --forbidden 'src/db/migrations/**' \
  --gate 'npm test -- --grep auth' \
  --evidence local \
  --json
```

The helper records the current integration `HEAD` as `baseCommit` unless
`--base-commit` is provided. It preserves task ID, package ID, optional thread
ID, pending worktree setup ID, expected branch, allowed/forbidden write set,
gates, and evidence-label expectations in the ledger.

If the worker worktree already exists, the older low-level command still works:

```bash
codex-orchestrator record-task \
  --id API-AUTH-LOCAL \
  --title "Auth endpoint implementation" \
  --package-id AUTH-PACKAGE \
  --thread-id optional-thread-id \
  --worktree /absolute/path/to/worktree \
  --branch codex/api-auth \
  --allowed 'src/auth/**' \
  --forbidden 'src/db/migrations/**' \
  --gate 'npm test -- --grep auth' \
  --evidence local
```

The helper stores `pendingWorktreeId` as an opaque string only. It does not
query Codex App, create sessions, create worktrees, merge, push, or clean up.
Pending setup is not active work. Until the real worktree path and branch are
known, `observe`, `status`, and heartbeat reports keep the task in
`pendingSetup` with local/static evidence.

After local git truth exists, reconcile the same task to the real worktree and
branch:

```bash
codex-orchestrator dispatch reconcile \
  --task-id API-AUTH-LOCAL \
  --json
```

`dispatch reconcile` uses local `git worktree list --porcelain` truth. It can
resolve by the branch already stored on the ledger task, or by an explicit
`--branch` / `--worktree` flag. The resolved worktree is still setup evidence
only; it is not proof that the task is correct or ready to merge.

## Runtime Status Snapshot

Use `status` when you want the shortest answer to "what is happening now" from
local ledger, repo, and worktree truth:

```bash
codex-orchestrator status
codex-orchestrator status --json
```

The runtime status surface stays `local/static`. It does not query Codex App
runtime APIs or claim direct daemon/session proof. It groups current work into
useful buckets when possible:

- `activeWorkers`
- `pendingSetup`
- `dirtyUncommitted`
- `completedUnreviewed`
- `blockers`
- `cleanupNeeded`
- `recentMergedOrCleaned`
- `availableDispatchSlots`

The same `status` / `observe` reports also include a `jobSummary` block for a
jobs/status-style view: total task count, per-status counts, and compact task
rows with id, package ID, status, signal, branch, pending worktree id, latest
timestamp, and next action. When related workers are recorded with
`--package-id`, reports also include `packageSummary`: package-level
active/review/blocked/cleanup state, member task counts, and the next suggested
package action. This is still local/static ledger and git evidence; it does not
attach to live Codex App sessions.

Markdown and HTML status outputs start with an "At a Glance" section for humans:
integration cleanliness, current package lane, review/blocker/cleanup pressure,
dispatch slots, and the first suggested action. Detailed runtime and job tables
remain below that summary.

Reports include a `projectMap` block as a lightweight onboarding signal. The
helper checks common files such as `docs/CODEBASE_MAP.md`,
`docs/project-map.md`, and `docs/architecture.md`. If none exists, the
recommended action is to ask Codex App to generate or read a concise project map
before first orchestration: module boundaries, owner docs, test commands,
shared contracts, and high-risk paths.

Each observation and runtime status item also includes a structured `state`
object so callers do not need to parse notes. The fields are local/static:

- `lifecycle`: the helper status, such as `pending-setup`, `active`,
  `completed-unreviewed`, `blocked`, `cleanup-needed`, `merged`, or `cleaned`.
- `setup`: whether setup is still an opaque `pending-worktree-id`, a missing
  recorded worktree, or a present worktree.
- `worktree`: whether the worktree path is not recorded, missing, recorded, or
  present.
- `branch`: whether the expected branch is matched, mismatched, detached, not
  recorded, or not inspected.
- `diff`: whether git truth shows dirty uncommitted work, a clean task commit,
  a clean branch with no task commit, or an unknown base comparison.
- `review`: whether orchestrator review is required, not ready, accepted,
  blocked, or terminal.
- `cleanup`: whether cleanup is needed, complete, not needed, or not inspected.

Git/worktree truth wins over advisory thread state. A worker thread that still
looks active but has a clean commit after `baseCommit` is reported as
`completed-unreviewed`; a worktree on detached `HEAD` while a branch is recorded
is `blocked`; dirty uncommitted work stays separate from clean committed work.

## Observe State

Use `observe` for one-shot reconciliation:

```bash
codex-orchestrator observe
codex-orchestrator observe --json
codex-orchestrator observe \
  --write-report .codex-orchestrator/heartbeat-report.json \
  --write-summary .codex-orchestrator/heartbeat-summary.md
```

The report includes:

- integration checkout dirty/error state,
- per-task observations,
- `runtimeStatus` with a compact local/static "what is happening now" summary,
- `jobSummary` with jobs/status-style counts and compact task rows,
- `projectMap` with local project-map readiness and a recommended first-run
  action,
- `overallStatus`,
- per-status `counts`,
- `reviewPressure` with active/review/stale/blocked/cleanup queues,
- `budgetSummary` and local/static `budgetPressure` warnings,
- recommended next actions for the orchestrator.

Budget pressure is helper-level evidence only. Missing task budgets and missing
routine spec budgets become warnings. Runtime budget pressure is computed from
recorded task timestamps in the local ledger. Review budget pressure is computed
only when the ledger records a `completed-unreviewed`/review-ready timestamp; if
the task is review-ready but that timestamp is absent, the report says the
review elapsed time is unknown instead of inventing it.

Important statuses:

| Status | Meaning |
|--------|---------|
| `quiet` | Keep monitoring; active work is within concurrency limit |
| `dispatch-possible` | Capacity is free and the repo is clean |
| `pending-setup` | Codex App has a pending setup ID or the expected worktree path does not exist yet |
| `review-needed` | A worker has a clean commit after `baseCommit` |
| `cleanup-needed` | A terminal task still has a worktree/branch that should be cleaned |
| `stale` | A task needs inspection or same-task nudge |
| `blocked` | Setup, branch, git, or integration state blocks safe progress |

## Run A Heartbeat

One check that writes report files and appends a heartbeat event:

```bash
codex-orchestrator heartbeat \
  --count 1 \
  --write-report .codex-orchestrator/heartbeat-report.json \
  --write-summary .codex-orchestrator/heartbeat-summary.md
```

Run continuously:

```bash
codex-orchestrator heartbeat \
  --interval 5m \
  --count 0 \
  --write-report .codex-orchestrator/heartbeat-report.json \
  --write-summary .codex-orchestrator/heartbeat-summary.md
```

`--count 0` means forever. Use your normal process manager, terminal multiplexer,
`launchd`, cron, or a Codex automation to run it.

Heartbeat Markdown and JSON include the same local/static budget pressure
warnings as `observe`. These warnings are for coordinator attention; they do not
start, stop, prioritize, or reschedule work by themselves.

Future budget-policy work should stay review-only until a human-approved policy
exists. The helper may report metadata gaps, near/exceeded local thresholds, and
unknown timing evidence, but dispatch, pause, merge, cleanup, or worker-control
decisions remain with the Codex App orchestrator and human reviewer.

## macOS External Watchdog

For hands-off runs, Codex App heartbeat is still the primary orchestrator
wakeup. The helper can add an OS-level warning layer so a missed App heartbeat
is visible the next time the Mac is awake and `launchd` runs.

Install a per-project user LaunchAgent:

```bash
REPO=/path/to/project ./scripts/install-macos-watchdog.sh
```

Optional environment variables:

```bash
REPO=/path/to/project \
BIN="$HOME/.local/bin/codex-orchestrator" \
INTERVAL=20m \
MISSED_AFTER=45m \
START_INTERVAL_SECONDS=1200 \
NOTIFY=1 \
SAY=0 \
./scripts/install-macos-watchdog.sh
```

Check the LaunchAgent plist, loaded status, and last local/static watchdog
report:

```bash
codex-orchestrator watchdog status --repo /path/to/project
codex-orchestrator watchdog status --repo /path/to/project --json
```

The LaunchAgent runs `scripts/macos-watchdog-run.sh`, which performs one
`heartbeat --count 1` check, writes:

- `.codex-orchestrator/watchdog-heartbeat-report.json`
- `.codex-orchestrator/watchdog-heartbeat-summary.md`
- `.codex-orchestrator/launchd-watchdog.out.log`
- `.codex-orchestrator/launchd-watchdog.err.log`

If the report contains `heartbeatStatus.status=missed`, the runner sends a
macOS notification. Set `SAY=1` during installation if voice notification is
wanted.

This watchdog is intentionally conservative. It does not create Codex sessions,
dispatch workers, review, merge, push, cleanup, or keep a sleeping Mac awake.
Its evidence is `local/static`: it can show that heartbeat checks were missed,
but not prove whether the cause was Codex App automation delivery, machine
sleep, OS power state, or thread scheduling.

## Run Read-Only Routines

The helper can emit local/static routine reports without mutating the ledger,
git state, worker sessions, releases, or worktrees:

```bash
codex-orchestrator run-routine pr-reviewer --task-id TASK
codex-orchestrator run-routine stale-task-rescuer --task-id TASK
codex-orchestrator run-routine ci-fixer --task-id TASK
codex-orchestrator run-routine release-verifier --tag v0.3.0-alpha.1
codex-orchestrator run-routine docs-drift-checker
codex-orchestrator run-routine evidence-label-auditor
codex-orchestrator run-routine orchestration-policy-auditor
codex-orchestrator run-routine roadmap-next-task-suggester
codex-orchestrator run-routine budget-policy-report
```

`pr-reviewer` includes the automated review checklist surface for merge review
of a ledger task. It stays read-only and local/static: it verifies task,
worktree, branch, dirty status, commits after `baseCommit`, `git diff
--name-status`, `git diff --check`, ledger allowed/forbidden path boundaries
when `writeSet` is recorded, locally detectable review/self-review/artifact/
evidence-label filename signals, and suggested narrow gates from the ledger
task. The checklist can fail clear path-boundary violations and warn on missing
local signals, but it does not merge, push, clean, dispatch, or replace human
review.

`docs-drift-checker` stays local/static. It compares runnable routines,
`routines/*.json`, key docs, and roadmap status text, then scans
`docs/reviews/*.md` for accepted or merged central-impact task notes that
mention command/routine/source changes without a central docs update or
explicit docs-drift decision. It reports those as `local` post-merge
docs-drift guard warnings only; it does not mutate git, ledger, worktrees,
sessions, releases, or external systems.

`evidence-label-auditor` stays local/static. It scans repo docs, review and
handoff notes, routine specs, routine reports, and ledger-shaped JSON for
deterministic `ELA001`-`ELA010` findings, including weak local/static/proxy
wording promoted to direct/pre/prod/device/runtime/payment proof without
explicit direct evidence wording. It does not inspect live devices, production,
pre, payment terminals, or runtime services, and it does not claim direct proof.

The budget-policy report surface is now runnable:

```bash
codex-orchestrator run-routine budget-policy-report \
  --write-report /tmp/budget-policy-report.json
```

It stays local/static: it reads roadmap and routine docs, routine specs,
optional repo-local ledger state, and an optional heartbeat report when those
files are already present. Its report keeps metadata coverage, local/static
pressure warnings, unknown timing states, and human-review recommendations
separate without introducing budget enforcement. It does not dispatch,
schedule, prioritize, pause, kill, merge, push, delete, clean worktrees, mutate
the ledger, or make budget eligibility decisions.

## Record Review/Merge/Cleanup Outcomes

The orchestrator should append events when it reviews, merges, rejects, blocks,
or cleans up a task:

```bash
codex-orchestrator append-event \
  --task-id API-AUTH-LOCAL \
  --type review \
  --status completed-unreviewed \
  --note "Clean commit exists; orchestrator review required."
```

After a successful merge and cleanup:

```bash
codex-orchestrator append-event \
  --task-id API-AUTH-LOCAL \
  --type cleanup \
  --status merged \
  --note "Merged to main, pushed, removed worktree, deleted local branch."
```

## Codex App Orchestrator Prompt

Use this shape when starting a fresh orchestrator:

```text
Use $codex-orchestrator for this repository.

Before dispatching, run `codex-orchestrator status` and
`codex-orchestrator observe --json` if the helper is installed. Use the ledger
and git/worktree truth as durable state, not stale chat memory.

For each new worker session, record it with
`codex-orchestrator dispatch record` including task ID, thread ID if available,
package ID, pending worktree setup ID, expected branch, base commit,
allowed/forbidden write set, gates, and evidence label expectations. After
local git worktree truth exists, run
`codex-orchestrator dispatch reconcile --task-id TASK` to write the resolved
worktree and branch back to the ledger.

During monitoring, use `codex-orchestrator heartbeat --count 1` or
`codex-orchestrator observe --json` to classify tasks. Review completed branches
before merge. After review/merge/reject/cleanup, append an event with
`codex-orchestrator append-event`.

Do not let the helper create sessions, merge, push, delete branches, or relabel
evidence. The Codex App orchestrator owns those decisions.
```

## Validate Routine Contracts

V2.5 adds routine contract validation. Routine specs are JSON files under
`routines/` and describe reusable workflow contracts such as stale rescue, PR
review, and CI fixing.

```bash
codex-orchestrator validate-routines --dir routines
codex-orchestrator validate-routines --dir routines --json
```

The validator checks that every routine declares inputs, allowed and forbidden
actions, gates, strict evidence labels, escalation rules, and a common output
shape. It does not execute the routine or create Codex App sessions.

After a routine runs, record its outcome in the ledger:

```bash
codex-orchestrator record-routine-run \
  --routine pr-reviewer \
  --task-id API-AUTH-LOCAL \
  --status passed \
  --evidence-local "go test ./..." \
  --action "reviewed diff and forbidden paths" \
  --next "merge task branch"
```

For blocked runs, include `--blocked-reason` and put the missing proof under
`--evidence-blocked`. The command records both `routineRuns[]` in the ledger and
a `routine-run` event in `events.jsonl`.

For richer output, write a report file that matches the routine output schema
and record it directly:

```bash
codex-orchestrator record-routine-run \
  --report-json examples/routine-reports/pr-reviewer.passed.json
```

`status`, `observe --json`, and heartbeat reports include the most recent
routine runs plus the runtime status buckets above, so a fresh orchestrator can
see both the latest proof/checker output and the current worker/worktree state
without scanning `events.jsonl` manually.

## Recovery From A Fresh Session

When a long orchestrator thread gets stale or compressed:

1. Start a fresh Codex App orchestrator session.
2. Read the project rules and progress docs.
3. Run `codex-orchestrator observe --json`.
4. Inspect any `review-needed`, `stale`, or `blocked` items.
5. Continue from repo/ledger truth instead of old task IDs in chat.

This is the main v2 improvement: the loop can restart without losing the task
state machine.
