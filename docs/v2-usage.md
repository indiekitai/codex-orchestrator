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
- `overallStatus`,
- per-status `counts`,
- `reviewPressure` with active/review/stale/blocked/cleanup queues,
- recommended next actions for the orchestrator.

Important statuses:

| Status | Meaning |
|--------|---------|
| `quiet` | Keep monitoring; active work is within concurrency limit |
| `dispatch-possible` | Capacity is free and the repo is clean |
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
Use $delegated-session-orchestrator for this repository.

Before dispatching, run `codex-orchestrator status` and
`codex-orchestrator observe --json` if the helper is installed. Use the ledger
and git/worktree truth as durable state, not stale chat memory.

For each new worker session, record it with `codex-orchestrator record-task`
including task ID, thread ID if available, worktree, branch, base commit,
allowed/forbidden write set, gates, and evidence label expectations.

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
routine runs so a fresh orchestrator can see the last reviewer/fixer/proof
outcome without scanning `events.jsonl` manually.

## Recovery From A Fresh Session

When a long orchestrator thread gets stale or compressed:

1. Start a fresh Codex App orchestrator session.
2. Read the project rules and progress docs.
3. Run `codex-orchestrator observe --json`.
4. Inspect any `review-needed`, `stale`, or `blocked` items.
5. Continue from repo/ledger truth instead of old task IDs in chat.

This is the main v2 improvement: the loop can restart without losing the task
state machine.
