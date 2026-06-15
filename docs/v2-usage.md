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

## Update An Existing Local Install

First install and later updates should feel the same: give Codex App the GitHub
repository and let it refresh the installed skill/helper. The friendliest path
is to ask Codex App to handle it:

```text
Please update my local codex-orchestrator installation from
https://github.com/indiekitai/codex-orchestrator.

Check the installed skill at ~/.codex/skills/codex-orchestrator and the helper
binary on PATH. Fetch or clone the latest repository if needed, update the
Codex App skill, rebuild the Go helper only if it is already installed or
clearly useful, and do not touch any project .codex-orchestrator/ledger.json
files. After updating, run a smoke check and tell me what changed.
```

For direct command-line use:

```bash
# Update the installed Codex App skill. If the helper already exists in
# ~/.local/bin, rebuild it too.
codex-orchestrator self-update

# Force helper rebuild as well.
codex-orchestrator self-update --with-helper

# Only sync ~/.codex/skills/codex-orchestrator.
codex-orchestrator self-update --skill-only
```

`codex-orchestrator self-update` runs `scripts/update-local.sh` from the chosen
source checkout or installed skill directory. It intentionally does not run
mutate project ledgers, dispatch sessions, merge, push, or clean worktrees.
Pull or download the repository version you want first, then run the command to
refresh your local Codex skill and optional helper.

If there is no local checkout, the helper can fetch this repository into a local
cache before updating:

```bash
codex-orchestrator self-update --from-github
```

That mode may run `git clone`/`git fetch` only inside its update cache. It still
does not mutate the project being orchestrated.

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

For a first project setup, ask Codex App to create starter planning files too:

```bash
codex-orchestrator init --write-templates
```

This adds non-overwriting templates under `.codex-orchestrator/`:

```text
.codex-orchestrator/orchestration-policy.md
.codex-orchestrator/package-plan.md
.codex-orchestrator/project-map.md
.codex-orchestrator/thread-map.md
.codex-orchestrator/pulse-threads.md
.codex-orchestrator/concepts.md
.codex-orchestrator/inbox.md
```

Use them to record the current product lane, feature package outcome, safe
worker queue, stop conditions, blocked external proof, project map, long-lived
Codex thread topology, stable concepts, and intake items before the first
hands-off run. Existing files are preserved unless
`--force` is explicitly used.

`thread-map.md` is for durable Codex App thread roles such as Router, Inbox,
Pulse, Log, and Project Orchestrator. `pulse-threads.md` contains reusable
prompt shapes for recurring read-only pulse checks, input triage, routing, and
decision logs. These files are local/static coordination state: verify live
thread ids and automations before taking irreversible action.

`concepts.md` is a local concept library for glossary terms, stable rules,
prior decisions, historical pitfalls, blocked concepts, and source docs.
`inbox.md` is a local intake surface for issues, user feedback, external
reviews, pulse outputs, and run observations before they become task contracts.
They are not task ledgers, external knowledge-base sync, or direct proof.

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
active/review/blocked/cleanup state, member task counts, completion progress,
external review status, and the next suggested package action. This is the
dashboard layer for "one feature package at a time": it should show whether a
lane is still running, waiting for review, blocked, or ready to close before
the orchestrator fills capacity with unrelated work. Package rows also expose
the local/static review policy decision. When a package has enough related
workers or matches higher-risk package keywords, `packageSummary` tells the
orchestrator that a package review pack and imported reviewer evidence are
needed before closeout. Reports also include `packageLaneGuard`, which warns
about ungrouped workers, multiple active package lanes, and available slots that
should only be used for the current package. A separate
`dispatchRecommendation` block is the action signal: `recommended=false` means
do not dispatch even when raw `availableSlots` is greater than zero, and
`reason` / `nextAction` explain whether to wait, reconcile setup, review, clean
up, or continue inside the same package lane. Reports also expose
`capacityOnly=true` and `capacityWarning` so UIs and agents do not treat
`availableSlots` as permission to dispatch unrelated filler work. A compact
`timeline` gives the recent task/routine sequence without reading raw ledger
events. This is still local/static ledger and git evidence; it does not attach
to live Codex App sessions.

Markdown and HTML status outputs start with a human-first `当前进度` section,
not raw ledger fields. It answers the questions a project owner usually has:
whether the run is healthy, which feature package is the current lane, what was
recently completed, what is running now, whether the human needs to act, the
next safe step, and the evidence boundary. Detailed runtime, package, and job
tables remain below that summary for the orchestrator or reviewer. The package
section is deliberately closer to a product dashboard than a raw job table: it
shows progress like `3/5 worker 已收口`, external review status, waiting queues,
and a package-specific next action. The HTML and Markdown surfaces also include
Preflight, Lane Guard, and Timeline sections so the owner can quickly answer
"can I walk away?", "are we still in one product module?", and "what happened
recently?"
When the ledger run mode is `drain` or `paused`, HTML status renders dispatch
slots as "do not dispatch" even when the raw available slot count is greater
than zero. Untracked `.codex-orchestrator/` files are separated as local
orchestration state, not business-code dirty status.

Reports include a `projectMap` block as a lightweight onboarding signal. The
helper checks common files such as `docs/CODEBASE_MAP.md`,
`docs/project-map.md`, and `docs/architecture.md`. If none exists, the
recommended action is to ask Codex App to generate or read a concise project map
before first orchestration: module boundaries, owner docs, test commands,
shared contracts, and high-risk paths.

Reports also include a `threadMap` block. The helper checks
`.codex-orchestrator/thread-map.md`, `docs/thread-map.md`, and
`THREAD_MAP.md`. If none exists, preflight warns before relying on multiple
long-lived Codex App threads, routers, inboxes, or pulse monitors. A thread map
does not prove that a thread or automation is alive; it only keeps the intended
topology out of chat memory.

Reports also include `concepts` and `inbox` blocks. The helper checks
`.codex-orchestrator/concepts.md`, common docs glossary/concepts files,
`.codex-orchestrator/inbox.md`, and common inbox files. Missing files are
local/static onboarding warnings: they mean the router or orchestrator may be
depending on chat memory for project vocabulary, prior decisions, feedback, or
external review intake.

Older ledgers may contain completed tasks that were recorded before
`packageId` existed. Those terminal ungrouped tasks remain available in JSON
`jobSummary.rows`, but `status` also exposes:

- `legacyTerminalUngrouped`: old cleaned/merged/rejected/abandoned tasks with
  no package id.
- `visibleRows`: current-action rows that still matter for the next decision.
- `ungroupedNonTerminal`: still-active or still-unreviewed tasks with no
  package id.

Only `ungroupedNonTerminal` should trip the package-lane guard. This keeps
legacy history from making a fresh status page look scattered while preserving
the full ledger for audit.

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
Git path commands are run with `core.quotePath=false` so non-ASCII paths remain
human-readable for path-boundary checks and review reports.

## Observe State

Use `observe` for one-shot reconciliation:

```bash
codex-orchestrator observe
codex-orchestrator observe --json
codex-orchestrator observe \
  --write-report .codex-orchestrator/heartbeat-report.json \
  --write-summary .codex-orchestrator/heartbeat-summary.md
```

## Package Closeout Status

Use `pack status` when a feature package has several worker commits and the
orchestrator needs a compact answer to "can this package close?"

```bash
codex-orchestrator pack status --package-id CHECKOUT-COUPONS --json
codex-orchestrator pack status \
  --package-id CHECKOUT-COUPONS \
  --write-report .codex-orchestrator/reviews/CHECKOUT-COUPONS-status.json
```

`pack status` embeds the existing `pack acceptance` report and package summary.
It can say:

- `ready-for-orchestrator-acceptance`: local/static package evidence is ready
  for the Codex App orchestrator's separate accept/reject/block decision.
- `external-review-needed`: review policy requires an imported reviewer signal
  before package closeout.
- `not-ready`: active, blocked, attention-needed, or cleanup-needed work
  remains.
- `blocked` or `reject-for-fixup`: local/static acceptance inputs are missing
  or failing.

It still does not merge, push, cleanup, deploy, dispatch, or produce direct
runtime/device/provider proof.

The report includes:

- integration checkout dirty/error state,
- per-task observations,
- `runtimeStatus` with a compact local/static "what is happening now" summary,
- `jobSummary` with jobs/status-style counts and compact task rows,
- `projectMap` with local project-map readiness and a recommended first-run
  action,
- `threadMap`, `concepts`, and `inbox` with local/static readiness signals for
  long-lived thread topology and project knowledge intake,
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

## Package Review And Acceptance

Use package-level review at a feature-package boundary, not after every tiny
worker. The normal sequence is:

```bash
codex-orchestrator pack review \
  --package-id PKG \
  --output .codex-orchestrator/review-pack/PKG

codex-orchestrator review run \
  --package-id PKG \
  --reviewer pi \
  --pack .codex-orchestrator/review-pack/PKG \
  --write-report .codex-orchestrator/review-pack/PKG/pi-review.json

codex-orchestrator pack acceptance \
  --package-id PKG \
  --write-report .codex-orchestrator/review-pack/PKG/package-acceptance.json
```

If `--task-id` is omitted, `pack review` and `pack acceptance` select tasks
recorded with the given `packageId`. This keeps the orchestrator from manually
copying a long list of task ids out of chat. External reviewer output is
`proxy/advisory`; the package acceptance report is `local/static`. Neither one
merges, pushes, cleans worktrees, deploys, or produces direct runtime/device/
provider proof by itself.

If a selected task has already been accepted, merged, pushed, and cleaned,
the worker worktree may be gone. `pack acceptance` treats terminal
`merged` / `released` / `cleaned` ledger tasks in post-cleanup mode: the report
uses terminal ledger state and recorded gates as local/static evidence, labels
fresh worktree diff proof as unavailable, and tells the orchestrator to rerun
integration gates when fresh proof is needed. A removed worktree after cleanup
is no longer a package-acceptance failure by itself.

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

Use `health` when you want one compact local/static snapshot that wraps the
same surfaces people usually check by hand:

```bash
codex-orchestrator health --repo /path/to/project
codex-orchestrator health --repo /path/to/project \
  --write-report .codex-orchestrator/health.json \
  --write-summary .codex-orchestrator/health.md
```

`health` aggregates repo cleanliness, runtime queue pressure, dispatch
recommendation, preflight, watchdog, project map, thread map, concepts, inbox,
and trust-risk state. It is intentionally read-only: it does not dispatch
Codex App sessions, mutate project ledgers, merge, push, deploy, cleanup, or
prove App wake delivery. Use `--fail-on-warning` only when you want a shell
gate; warnings remain `local/static` evidence.

Before leaving a project unattended, run the one-shot local/static preflight:

```bash
codex-orchestrator preflight --repo /path/to/project
codex-orchestrator preflight --repo /path/to/project \
  --write-report .codex-orchestrator/preflight.json \
  --write-summary .codex-orchestrator/preflight.md
```

`preflight` checks repo cleanliness, ledger shape, dispatch mode, recent
heartbeat gap, watchdog status, project-map/thread-map/concepts/inbox presence,
package-lane health, and missing external-review evidence. A warning does not prove Codex App or OS
failure; it is a local/static signal to surface before an unattended run.
Warnings exit successfully by default so preflight can write status artifacts
without failing the monitor turn. Add `--fail-on-warning` when using it as a
shell gate. The default heartbeat assumptions are `--interval 20m` and
`--missed-after 45m`.

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
`heartbeat --check-only --count 1` check, writes:

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

The `--check-only` part is important. The App heartbeat turn may append a
normal heartbeat event after Codex App wakes the thread. The external watchdog
must not append that same event, because doing so would make the local ledger
look fresh even when Codex App did not actually wake the orchestrator thread.

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

## Turn Failures Into Eval Fixtures

When a real run exposes a repeatable orchestration failure, treat the worker or
review note as Maker output and produce a Checker draft first:

```bash
codex-orchestrator eval draft-failure \
  --id heartbeat-prompt-churn \
  --from-review docs/reviews/heartbeat-review.md \
  --file docs/reviews/heartbeat-review.md \
  --expect OPA006=1 \
  --write-report /tmp/heartbeat-eval-draft.json
```

The draft report is local/static evidence. It shows the actual rule hits,
whether they match the expected `OPAxxx` counts, the fixture JSON that would be
written, and the suggested `eval add-failure` command. It does not write the
fixture.

After a human or orchestrator checker accepts the draft, lock it into the suite:

```bash
codex-orchestrator eval add-failure \
  --id heartbeat-prompt-churn \
  --text-file docs/reviews/heartbeat-review.md \
  --file docs/reviews/heartbeat-review.md \
  --expect OPA006=1
```

Then run:

```bash
codex-orchestrator eval run --repo .
codex-orchestrator policy check --repo .
```

This is the conservative self-improvement loop: observe a failure, draft a
regression, review it, then lock it. The helper does not automatically change
policy rules, merge code, dispatch workers, or claim runtime proof.

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

When local git truth is more current than the ledger, run:

```bash
codex-orchestrator observe --reconcile --write --json
```

This writes only deterministic local/static reconciliation back to the ledger:
resolved worktree/branch for pending setup and `completed-unreviewed` for a
clean task commit after `baseCommit`. It does not accept, merge, push, cleanup,
release, deploy, or prove runtime behavior.

For external watchdogs, use `codex-orchestrator heartbeat --check-only --count 1`
so the watchdog can write reports without appending a heartbeat event.

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
