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

## Record A Worker Session

After the Codex App orchestrator creates a worker session/worktree, record it:

```bash
codex-orchestrator record-task \
  --id API-AUTH-LOCAL \
  --title "Auth endpoint implementation" \
  --thread-id optional-thread-id \
  --worktree /absolute/path/to/worktree \
  --branch codex/api-auth \
  --allowed 'src/auth/**' \
  --allowed 'tests/auth/**' \
  --forbidden 'src/db/migrations/**' \
  --gate 'npm test -- --grep auth' \
  --evidence local
```

The helper records the current integration `HEAD` as `baseCommit` unless
`--base-commit` is provided.

If Codex App returns an opaque pending worktree setup ID before the worktree
exists, record the task immediately without inventing a path:

```bash
codex-orchestrator record-task \
  --id API-AUTH-LOCAL \
  --title "Auth endpoint implementation" \
  --thread-id optional-thread-id \
  --pending-worktree-id pending-worktree-id-from-codex-app \
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

After the actual worktree and branch are known, reconcile the same task with an
event:

```bash
codex-orchestrator append-event \
  --task-id API-AUTH-LOCAL \
  --type setup-complete \
  --status active \
  --worktree /absolute/path/to/worktree \
  --branch codex/api-auth \
  --note "Codex App worktree setup completed."
```

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

For each new worker session, record it with `codex-orchestrator record-task`
including task ID, thread ID if available, worktree, branch, base commit,
allowed/forbidden write set, gates, and evidence label expectations. If Codex
App only returns a pending worktree setup ID, record `--pending-worktree-id`
immediately, then append a setup event with `--worktree` and `--branch` after
the setup completes.

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
