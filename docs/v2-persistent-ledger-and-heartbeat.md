# v2 Persistent Ledger And Heartbeat

`codex-orchestrator` v1 is a supervised Codex skill: the orchestrator thread
keeps the working state in chat, repo docs, git branches, and active worktrees.

v2 adds a durable state layer. The goal is not to replace engineering review or
turn the skill into an unsupervised agent OS. The goal is to make the loop
recoverable across thread restarts, stale automations, and overnight runs.

## Scope

v2 should provide:

- a persistent task ledger,
- append-only heartbeat observations,
- deterministic stale-task classification,
- explicit merge/review states,
- enough repo truth to resume orchestration from a fresh session,
- and a safe read-only heartbeat checker that can be run by cron, launchd, or a
  Codex automation.

v2 should not initially:

- create Codex sessions by itself,
- push or merge without a reviewing orchestrator,
- overwrite dirty work,
- treat thread status as authoritative,
- or claim direct/runtime proof from local git state.

## Ledger File

The recommended ledger path is project-local and untracked by default:

```text
.codex-orchestrator/ledger.json
```

For shared teams, commit a template instead:

```text
.codex-orchestrator/ledger.example.json
```

The ledger stores durable state. It should be small enough to inspect in a code
review or paste into a new orchestrator session.

Required top-level fields:

| Field | Purpose |
|-------|---------|
| `version` | Ledger schema version |
| `projectRoot` | Absolute or repository-relative root |
| `defaultBranch` | Integration branch, usually `main` |
| `remote` | Remote name, usually `origin` |
| `pushPolicy` | `manual`, `normal`, or project-specific |
| `maxConcurrency` | Default active implementation sessions |
| `createdAt` / `updatedAt` | ISO timestamps |
| `tasks` | Task records |

Each task should include:

| Field | Purpose |
|-------|---------|
| `id` | Stable task ID |
| `title` | Short human-readable name |
| `threadId` | Delegated session/thread ID, if available |
| `worktree` | Worker worktree path |
| `branch` | Task branch |
| `baseCommit` | Commit used to start the task |
| `status` | Current orchestrator state |
| `writeSet.allowed` | Allowed paths |
| `writeSet.forbidden` | Forbidden paths |
| `gates` | Required commands or checks |
| `evidence` | Expected proof type and labels |
| `lastObservation` | Last heartbeat result |
| `history` | Short event list |

## Status Values

Use explicit states instead of overloaded words like "done":

| Status | Meaning |
|--------|---------|
| `pending-setup` | Worktree/session creation requested but not verified |
| `active` | Worker appears to be making progress |
| `stale-needs-inspection` | No fresh progress or thread/worktree mismatch |
| `completed-unreviewed` | Clean task commit exists; orchestrator review not done |
| `reviewing` | Orchestrator is checking diff/gates/docs/evidence |
| `merged` | Accepted and merged to the integration branch |
| `rejected` | Reviewed and not accepted |
| `blocked` | Cannot proceed without missing input, environment, or decision |
| `abandoned` | No useful scoped work to rescue |

## Heartbeat Observation

A heartbeat must be read-only. It may inspect:

- `git status --short --branch` in the integration checkout,
- `git worktree list --porcelain`,
- each task worktree's `git status --short --branch`,
- each task worktree's recent commits,
- thread status if the host exposes it,
- task final handoff text if available.

It must not:

- edit files,
- run destructive cleanup,
- merge,
- push,
- delete branches,
- or relabel evidence quality.

The heartbeat output should be one of:

- `quiet`: active tasks are still moving or no action is needed,
- `review-needed`: a clean task commit appears ready for review,
- `stale`: the task needs inspection or a targeted nudge,
- `blocked`: a setup, tool, environment, or human-action blocker exists,
- `dispatch-possible`: capacity is free and the roadmap has safe next tasks.

## Decision Rules

1. If the worktree is missing and the task is still `pending-setup`, keep it
   pending until the setup window expires, then mark `stale-needs-inspection`.
2. If the worktree is clean and has commits beyond `baseCommit`, classify it as
   `completed-unreviewed` even if the thread still looks active.
3. If the worktree has uncommitted changes and no recent history or final
   handoff, classify it as `stale-needs-inspection`.
4. If the branch does not match the expected branch, report a setup/blocker
   condition. Do not merge from the wrong branch.
5. If the integration checkout is dirty, do not dispatch or merge new work until
   the dirty state is understood.
6. If all accepted tasks are merged and capacity is free, a new orchestrator may
   dispatch the next package from the roadmap.

## Event History

The ledger should keep short events rather than full logs:

```json
{
  "at": "2026-06-10T09:30:00+08:00",
  "type": "heartbeat",
  "status": "completed-unreviewed",
  "note": "Clean commit exists in worktree; orchestrator review required"
}
```

Long artifacts, screenshots, logs, and review docs should live in the target
project, not inside the ledger.

## First Implementation Step

This repository includes a read-only helper:

```bash
python3 scripts/ledger_heartbeat.py observe --ledger examples/ledger.example.json
```

It does not create sessions, merge, push, or clean worktrees. It only compares
the ledger with local git truth and prints the next suggested orchestrator
action.

The core skill does not require Python. This repository now includes a Go CLI
seed for machines that should not depend on `python3`:

```bash
go build -o codex-orchestrator ./cmd/codex-orchestrator
./codex-orchestrator init
./codex-orchestrator record-task --id TASK --worktree /path/to/wt --branch codex/task
./codex-orchestrator observe --ledger examples/ledger.example.json
./codex-orchestrator append-event --type review --task-id TASK --status completed-unreviewed
```

The Python helper remains as a prototype and compatibility reference. If neither
helper is available, use the same ledger schema manually and let the Codex App
orchestrator inspect `git status`, `git worktree list`, and task worktrees
directly.

For compatibility, the original form still works:

```bash
python3 scripts/ledger_heartbeat.py --ledger examples/ledger.example.json
```

## Helper CLI

The helper CLI is intended for the Codex App orchestrator to call from a project
checkout. It is not another AI agent and it does not replace Codex App session
dispatch.

Initialize a project-local ledger:

```bash
python3 /path/to/codex-orchestrator/scripts/ledger_heartbeat.py init
```

Record a delegated task after the App orchestrator creates a worker session:

```bash
python3 /path/to/codex-orchestrator/scripts/ledger_heartbeat.py record-task \
  --id API-AUTH-LOCAL \
  --title "Auth endpoint implementation" \
  --thread-id optional-thread-id \
  --worktree /absolute/path/to/worktree \
  --branch codex/api-auth \
  --allowed 'src/auth/**' \
  --allowed 'tests/auth/**' \
  --forbidden 'src/db/migrations/**' \
  --gate 'npm test -- --grep auth'
```

Observe the ledger and local git/worktree truth:

```bash
python3 /path/to/codex-orchestrator/scripts/ledger_heartbeat.py observe
python3 /path/to/codex-orchestrator/scripts/ledger_heartbeat.py observe --json
python3 /path/to/codex-orchestrator/scripts/ledger_heartbeat.py observe \
  --write-report .codex-orchestrator/heartbeat-report.json
```

Summarize task states:

```bash
python3 /path/to/codex-orchestrator/scripts/ledger_heartbeat.py status
```

Append an event and optionally update a task:

```bash
python3 /path/to/codex-orchestrator/scripts/ledger_heartbeat.py append-event \
  --task-id API-AUTH-LOCAL \
  --type review \
  --status completed-unreviewed \
  --note "Clean commit exists; orchestrator review required."
```

Use it as a bridge between v1 and a future daemon: first make state durable,
then make the heartbeat repeatable, then add safe integrations one at a time.
