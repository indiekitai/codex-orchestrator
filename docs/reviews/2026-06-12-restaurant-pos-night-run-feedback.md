# Restaurant POS Rewrite Night Run Feedback

Date: 2026-06-12

This note records a read-only retrospective from a restaurant POS rewrite overnight
Codex App orchestration run. It is local/project feedback for
`codex-orchestrator`; it is not direct proof of Codex App automation internals,
production runtime, payment, hardware, or provider behavior.

Evidence label: `local/project-feedback`.

## Facts Reported By The Orchestrator

- The restaurant POS rewrite ledger had 20 tasks: 19 `cleaned` and 1 `blocked`.
- `main` matched `origin/main`; no `codex/*` task branches or worker worktrees
  remained.
- The heartbeat automation had been deleted by the time of the retrospective.
- A 2026-06-12 01:33 to 07:10 Asia/Shanghai gap occurred where the thread was
  not visibly woken, despite an intended 20-minute heartbeat schedule. The
  retrospective could not prove whether the cause was Codex App automation
  delivery, machine sleep, queue dispatch, or thread scheduling.
- One task, `TF-P2-POS-SPLIT-CHECK-UI-LOCAL-READINESS`, was `blocked` because
  worktree setup failed with `fatal: invalid reference`. The failure came from
  treating the desired new task branch as an existing starting reference.

## What Worked

- Durable ledger state was useful: task id, `pendingWorktreeId`, branch,
  worktree, allowed/forbidden paths, gates, status, and events survived beyond
  chat memory.
- `observe` helped identify active, dirty, completed-unreviewed, and cleanup
  states from local repo/worktree truth.
- Evidence labels prevented local/unit/build/proxy results from being promoted
  into pre/prod/device/payment proof.
- The review/merge/push/cleanup loop worked once tasks had real worktrees and
  clean task commits.

## Problems To Productize

### 0. What the 5-hour heartbeat gap was and was not

The heartbeat configuration itself was not five hours. It was intended to wake
the orchestrator every 20 minutes until 09:00 Asia/Shanghai. The visible record
showed an automatic wakeup around 01:33 and the next visible wakeup around
07:10, with no heartbeat message delivered to the thread in between.

The gap was not caused by:

- foreground `sleep` or long polling in the orchestrator turn;
- the orchestrator deciding the queue was drained;
- the orchestrator waiting for user confirmation;
- a worker-level blocker;
- the heartbeat being intentionally configured to a five-hour interval.

The defensible conclusion is narrower: Codex App heartbeat did not deliver
20-minute wakeups into that thread during the gap. The exact layer remains
`blocked`: App automation runner, queue delivery, machine sleep, operating
system power state, or thread scheduling cannot be proven from repo, ledger, or
automation file evidence alone.

The orchestrator responsibility was still real: it did not have a missed
heartbeat detection/reporting loop. A better status surface would have made the
07:10 wakeup say "missed about 5 hours of scheduled checks" before continuing
normal task processing.

### 1. Blocked setup must outrank pending setup

If a task has a `pendingWorktreeId` but setup later fails, the ledger status
must become `blocked`, and `observe` must not keep presenting the task as
`pending-setup`.

For `TF-P2-POS-SPLIT-CHECK-UI-LOCAL-READINESS`, the concrete setup error was:

```text
fatal: invalid reference: codex/TF-P2-POS-SPLIT-CHECK-UI-LOCAL-READINESS
git worktree add failed
```

This means the app tried to create a worktree from a git reference that did not
exist. The orchestrator had passed the desired new worker branch name as though
it were an existing starting branch/reference. The correct shape is to start
from `main` or a known base commit, then create or switch to the target worker
branch using the tool's new-branch semantics. No real worker worktree, task
branch, or code output existed for that task, so the only correct evidence label
was `blocked`.

Implemented follow-up:

- `inspectTask` now gives ledger `blocked` status precedence over
  pending/worktree hints.
- A regression test covers a failed pending setup with `fatal: invalid
  reference` and verifies it appears under blockers, not pending setup.

### 2. Stop/drain requests need ledger state

The orchestrator previously had to encode "finish current tasks but do not
dispatch more" in heartbeat/policy prose. That is fragile because `observe`
could still recommend `dispatch-possible`.

Implemented follow-up:

- Ledger now supports run-level `dispatchMode`: `active`, `drain`, or `paused`.
- `codex-orchestrator run-mode set --dispatch-mode drain|paused|active` records
  the mode in ledger and events.
- `observe` now returns `dispatch-draining` or `dispatch-paused` instead of
  recommending new dispatch when the run is drained or paused.

### 3. Heartbeat missed-run detection remains outside this helper

The 5-hour wakeup gap cannot be proven or fixed purely from repo-local state.
The helper can record heartbeat reports and show status snapshots, but it
cannot prove Codex App scheduler delivery, machine sleep, or thread wakeup
success without an App-level automation status API.

Recommended future surface:

- App automation status should expose target thread, schedule, last attempted
  wakeup, last successful turn, next due time, and missed-run count.
- Until that exists, orchestrators should record heartbeat reports when woken,
  and users should treat long gaps as `blocked` App automation evidence rather
  than repo or worker proof.

### 4. Feature-package continuity still matters

The retrospective confirmed that safe unattended work can degrade into a
scattered backlog sweep. A useful daily report needs one product lane at a time:
for example POS split/merge checkout readiness, Customer ordering/coupon
checkout, Z-report/Close Day, Staff/RBAC, or KDS routing.

Existing coverage:

- The skill already requires choosing a feature package before tasks.
- The policy/eval fixtures already cover unrelated safe-backlog dispatch.

Remaining improvement:

- A future package ledger should make package outcome, active workers, merge
  order, and blocked evidence first-class rather than relying on task names and
  human-readable policy notes.

## Boundaries

- `local`: repo/ledger/docs/test evidence from this repository.
- `blocked`: Codex App automation delivery internals, machine sleep, scheduler
  truth, and exact missed-wakeup cause.
- `blocked`: no direct restaurant POS rewrite runtime, payment, hardware, provider,
  pre/prod, or device proof is claimed here.
