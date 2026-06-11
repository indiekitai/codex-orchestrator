---
name: codex-orchestrator
description: Use when the user wants Codex App to run a supervised outer loop for multi-session development: plan bounded work, dispatch isolated worktree sessions, monitor with heartbeat and ledger truth, review/merge/push/cleanup, rescue stale tasks, and keep direct/proxy/local/blocked evidence honest.
---

# codex-orchestrator

## Assumptions And Configuration

This skill is workflow-oriented rather than project-specific. At the start of
each orchestration run, discover and record the local equivalents of these
values instead of assuming them:

- repository root and default branch (`main`, `master`, or project-specific),
- remote/push policy (`push <remote> <default-branch>` only when normal for the
  project or explicitly requested),
- available delegation surface (Codex App worktree sessions, another supported
  worker/subagent path, or no delegation available),
- available automation/checkback mechanism,
- human notification mechanism, if any,
- user's preferred language for human-action notifications.

If a project has no saved Codex App project/worktree support, do not pretend the
Codex App-specific setup steps are available. Use the supported delegation
surface for that environment, or report the tooling blocker before dispatching.

## Core Idea

Use this skill when the best move is not one big implementation in the current thread, but a controlled pipeline of small independent sessions. The orchestrator owns decomposition, dispatch, monitoring, review, merge, push, and cleanup.

This is a Codex App-first supervised outer loop. It is not a standalone daemon,
a fully autonomous agent operating system, or a replacement for engineering
judgment. Worker sessions still run their own edit/test/fix inner loops; this
skill manages the project-level outer loop around those workers.

This works best in early development or large-module buildout where many slices can move in parallel. It works poorly for hardware-heavy acceptance, production deploys, payment tests, or steps requiring frequent human observation; keep those serialized and explicit.

Treat this skill as a living runbook, not a frozen policy. When orchestration reveals a better rule, a repeated mistake, a misleading prompt pattern, or a new safety constraint, update the skill or the active automation promptly so future sessions inherit the correction. Do not only mention the improvement in chat and then keep operating from stale instructions.

## Operating Loop

1. Check the real repo state first:
   - `git status --short --branch`
   - `git worktree list`
   - recent delegated sessions / pending worktree setup
   - current roadmap/progress docs if present
2. Identify shared contract surfaces and serialize them before parallel work:
   - proto / API envelopes
   - DB migrations
   - Cloud API contracts
   - command/event handlers
   - terminal sync contracts
   - hardware/device ownership
   - If the repo has a contract sync gate that requires same-change consumers
     (for example a proto sync check requiring multiple consumers to update
     after `.proto` changes), do not dispatch an unmergeable "contract-only"
     branch. Keep the work serialized, but include the minimal required consumer
     compile/wiring updates in that same serial task, or stop with a blocker
     before editing.
3. Choose the next feature package before choosing the next small task:
   - When a domain already has multiple first closures, define the next
     module-level milestone first (e.g., "User Dashboard MVP" or
     "Inventory Management Admin MVP").
   - Break that milestone into the fewest serial/parallel worker contracts that
     can safely merge, instead of filling the queue with many small isolated
     slices.
   - Only dispatch a tiny task when it removes a named blocker, proves a needed
     runtime surface, or safely lands a shared contract needed by the larger
     milestone.
   - Keep a short package ledger: milestone outcome, dependency graph, active
     worker contracts, merge order, gates, and what evidence remains blocked.
4. Choose at most two active sessions by default, with a narrow option to raise
   to three after shared contracts are merged and the write sets are clearly
   disjoint:
   - one hardware or long-running task if needed
   - one non-hardware task with a disjoint write set
   - a third non-hardware task only when the default branch is clean, no
     contract/migration/API branch is active, and each task has an independent
     module/write set
   - never two tasks editing the same proto, migration, core aggregate, review file, or artifact root
5. Give each session a bounded task contract.
6. Monitor dynamically instead of by fixed task IDs.
7. When a session finishes, review before merging:
   - diff and file boundary
   - self-review present
   - contract/shared-surface conflicts
   - docs/progress/reviews/artifacts synchronized
   - gates run and credible
   - no evidence exaggeration
8. If accepted, merge to the default integration branch, push if requested or
   normal for this project, clean the worktree, and delete the local branch.
9. Before deleting or disabling the task-specific heartbeat, decide whether the
   orchestration loop itself should continue:
   - run `codex-orchestrator observe --json` or equivalent repo/thread checks,
   - inspect the roadmap/routine queue for the next safe task,
   - if capacity is available and safe work remains, dispatch the next bounded
     task or replace the task-specific heartbeat with a fresh next-task monitor,
   - if no safe work remains, record that queue-drained state and only then
     delete the heartbeat,
   - if the next task choice is blocked by missing context, notify with the
     blocker instead of silently stopping.
10. If rejected, report blocking findings and leave the branch/worktree for targeted fix or cleanup.

Task-specific heartbeats are watchdogs for the current child task, not the
whole orchestrator lifecycle. Completing one child task must not be treated as
permission to stop the larger loop when the user asked the orchestrator to keep
working through a queue or roadmap.

## Orchestrator State Ledger

After dispatching or discovering a delegated session, keep a compact ledger in the orchestrator thread or current status note. Do not rely on memory or stale automation text.

If the repository has the v2 helper installed, prefer a durable project-local
ledger over chat-only state:

```bash
codex-orchestrator init
codex-orchestrator record-task --id TASK --worktree /path/to/worktree --branch codex/task --max-runtime-minutes 90 --review-budget-minutes 25
codex-orchestrator observe --json
codex-orchestrator heartbeat --count 1 --write-report .codex-orchestrator/heartbeat-report.json --write-summary .codex-orchestrator/heartbeat-summary.md
codex-orchestrator append-event --task-id TASK --type review --status completed-unreviewed --note "Ready for orchestrator review."
```

The helper is not a session launcher and must not be treated as one. It is a
state and heartbeat tool. The Codex App orchestrator still owns worker dispatch,
review, merge, push, and cleanup decisions.

Optional task runtime/review budget metadata is visibility-only. `observe` and
`heartbeat` can surface recorded budgets, but the helper must not kill
processes, schedule sessions, or enforce budget decisions.

If the repository includes v2.5 routine contracts, validate them before relying
on routine names in a plan:

```bash
codex-orchestrator validate-routines --dir routines
```

Treat routines as workflow contracts, not magic commands. A routine can define
triggers, inputs, allowed actions, forbidden actions, gates, evidence labels,
escalation rules, and the output shape expected by the orchestrator. It does not
create Codex App sessions, merge, push, clean worktrees, or upgrade
local/proxy evidence into direct proof.

The helper includes conservative MVP runners for PR reviewer, stale task
rescuer, CI fixer, release verifier, docs drift checker, evidence label auditor,
and roadmap next-task suggester routines. Most runners are read-only. The
ci-fixer runner is different: it executes trusted gate commands already
recorded on a ledger task, so use it only when the ledger/gate source is
trusted.

```bash
codex-orchestrator run-routine pr-reviewer --ledger .codex-orchestrator/ledger.json --task-id TASK --write-report /tmp/pr-reviewer-report.json
codex-orchestrator run-routine stale-task-rescuer --ledger .codex-orchestrator/ledger.json --task-id TASK --write-report /tmp/stale-task-rescuer-report.json
codex-orchestrator run-routine ci-fixer --ledger .codex-orchestrator/ledger.json --task-id TASK --write-report /tmp/ci-fixer-report.json
codex-orchestrator run-routine release-verifier --tag v0.3.0-alpha.1 --write-report /tmp/release-verifier-report.json
codex-orchestrator run-routine docs-drift-checker --write-report /tmp/docs-drift-checker-report.json
codex-orchestrator run-routine evidence-label-auditor --write-report /tmp/evidence-label-auditor-report.json
codex-orchestrator run-routine orchestration-policy-auditor --write-report /tmp/orchestration-policy-auditor-report.json
codex-orchestrator run-routine roadmap-next-task-suggester --write-report /tmp/roadmap-next-task-suggester-report.json
codex-orchestrator policy check --write-report /tmp/policy-check-report.json
codex-orchestrator eval run --write-report /tmp/eval-run-report.json
codex-orchestrator eval add-failure --id dry-run-example --text "Dry run mode can dispatch workers immediately." --expect OPA001=1
```

The PR reviewer runner inspects only local git/static state from the ledger task worktree:
task existence, worktree existence, expected branch match, git status,
`git diff --name-status baseCommit..HEAD`, `git diff --check baseCommit..HEAD`,
and whether commits exist after `baseCommit`. Treat its evidence as `local`,
not `direct` runtime proof, and record the JSON report separately when the run
should become durable ledger truth.

The stale task rescuer runner is also read-only. It records ledger status, last
observation, recent task history, worktree/branch state, `git status --short
--branch`, `git log --oneline -3`, committed diff names, and uncommitted local
change evidence when present. It classifies clean committed work as `passed`
for orchestrator review, useful uncommitted work as `failed` with a same-worker
or same-task takeover next action, and missing worktree, branch mismatch,
missing `baseCommit`, or git inspection failures as `blocked`. Its MVP report
uses only `local` and `blocked` evidence; it does not stage, commit, merge,
clean, dispatch, update ledger status, or claim direct/proxy runtime proof.

The ci-fixer runner is a CI/local gate classifier, not an automatic code fixer:
it requires explicit trusted gates recorded on the ledger task, checks worktree
and branch state, refuses dirty worktrees, compares `baseCommit..HEAD`, records
committed file names, and runs recorded gate commands in the task worktree with
a local timeout. Because recorded gates are shell commands, do not run
ci-fixer against an untrusted repository or untrusted ledger. It classifies
passing gates with committed work as `passed`, dirty worktrees or failing gates
as `failed` with a same-worker or same-task takeover next action, and missing
gates, missing `baseCommit`, branch mismatch, or git inspection failures as
`blocked`. Its MVP report uses only `local` and `blocked` evidence; it does not
edit files, stage, commit, merge, push, clean, dispatch, update ledger status,
or claim direct/proxy runtime proof.

The release verifier runner is read-only and does not load or update the
ledger. It verifies a supplied local git tag, records the local tag object type,
reads GitHub release metadata through `gh release view` when `gh` is available,
checks alpha/beta/rc prerelease flags, and compares release asset names against
this repo's default Go CLI asset set or explicit repeated `--expected-asset`
values. It classifies missing tags, missing releases, drafts, prerelease
mismatches, and missing assets as `failed`; unavailable `gh`, auth/network
errors, or unparseable release metadata as `blocked`. Its MVP report uses
`local`, `proxy`, and `blocked` evidence; it does not create or edit releases,
move tags, upload assets, stage, commit, merge, push, clean, dispatch, mutate
the ledger, or claim production/runtime proof.

The docs drift checker runner is read-only and does not load or update the
ledger. It parses the local `run-routine` command surface from
`cmd/codex-orchestrator/main.go`, compares runnable routine IDs against
`routines/*.json`, and checks `README.md`, `README.zh-CN.md`, `SKILL.md`,
`docs/routines/README.md`, and `docs/roadmap.md` when present for obvious
missing references or stale status text. It classifies missing specs or docs
mentions as `failed`, missing repository/source/spec access as `blocked`, and a
clean static comparison as `passed`. Its MVP report uses only `local` and
`blocked` evidence; it does not stage, commit, merge, push, tag, release, clean
worktrees, dispatch sessions, mutate the ledger, or claim runtime proof.

The evidence label auditor runner is read-only and does not load or update the
ledger. It scans explicit repo-local docs, routine specs, routine report JSON,
and ledger-shaped JSON for obvious evidence-label issues: weak evidence labels
near overstated proof wording, missing
RoutineRunReport evidence buckets, and direct evidence recorded for routines
whose specs explicitly reserve direct evidence. It uses deterministic named
policy/eval rules (`ELA001`-`ELA009`), treats glossary/prohibition/blocked
definition wording as allowed negatives, and summarizes local rule hits when
findings appear. Findings are local/static suspicions until a reviewer confirms
them. Its MVP report uses only `local` and `blocked` evidence; it does not
stage, commit, merge, push, tag, release, clean worktrees, dispatch sessions,
mutate the ledger, or claim runtime proof.

The orchestration policy auditor runner is the first V4 policy/eval checker.
It is read-only and does not load or update the ledger. It scans repo-local
orchestration docs, prompts, routine specs, routine reports, and ledger/event
files for deterministic policy rules (`OPA001`-`OPA005`): dry-run dispatch
barrier, no-main-checkout fallback guard, heartbeat continuation guard,
delegated worker boundary, and evidence promotion boundary. Findings are
local/static suspicions until a reviewer confirms them. Its MVP report uses
only `local` and `blocked` evidence; it does not stage, commit, merge, push,
tag, release, clean worktrees, dispatch sessions, mutate the ledger, or claim
runtime proof.

Use `codex-orchestrator policy check` as the preferred V4 policy/eval entry
when you want the local orchestration policy scan plus the repo's eval
fixtures. The bundled fixtures live under `eval/orchestration-policy-auditor/`
and cover real orchestration failure classes: dry-run dispatch without
approval, setup-failure fallback into the orchestrator checkout, stopping the
larger queue after one child task, delegated worker prompts missing mandatory
boundaries, and evidence promotion from local/proxy/weak to direct. This
command is also read-only: it does not create sessions, mutate git, update the
ledger, or claim runtime proof.

Use `codex-orchestrator eval run` when you only want to run the fixture suite
without scanning the current repository text. The default suite is
`orchestration-policy-auditor`; it compares actual `OPAxxx` rule-hit counts
against each fixture's `expectedRuleHits`.

Use `codex-orchestrator eval add-failure` to add a manually supplied failure
case to the fixture suite. The MVP requires explicit `--text` or `--text-file`
and at least one `--expect RULE=N`. It validates the text against the current
rules before writing JSON and refuses to overwrite existing fixtures unless
`--force` is supplied.

The roadmap next-task suggester runner is read-only and does not mutate the
ledger. It parses remaining candidate tasks from `docs/roadmap.md`, compares
them against local runnable routine IDs and `routines/*.json`, optionally
filters duplicate active/pending/merged matches from a repo-local
`.codex-orchestrator/ledger.json`, and prefers conservative read-only local
tasks over mutating, release-scoped, or network-dependent work. If only unsafe
items remain, it reports a queue-drained next action instead of pretending to
dispatch. Its MVP report uses only `local` and `blocked` evidence; it does not
stage, commit, merge, push, tag, release, clean worktrees, dispatch sessions,
or claim runtime proof.

After a routine is actually run, record the outcome so future orchestrator
sessions can resume from ledger truth:

```bash
codex-orchestrator record-routine-run --routine pr-reviewer --task-id TASK --status passed --evidence-local "go test ./..." --action "reviewed diff" --next "merge task branch"
```

If the routine produced a JSON report, prefer recording the report directly:

```bash
codex-orchestrator record-routine-run --report-json examples/routine-reports/pr-reviewer.passed.json
```

For blocked routine runs, include `--blocked-reason` and at least one
`--evidence-blocked` item. Keep `direct`, `proxy`, `local`, and `blocked`
evidence separate.

Record:

- task ID and short outcome,
- thread ID,
- worktree path,
- branch,
- base commit,
- allowed/forbidden write set,
- hardware/env owner and expected release condition,
- current status: `active`, `pending setup`, `completed-unreviewed`, `merged`, `released`, `cleaned`, `rejected`, `abandoned`, or `blocked`,
- commit hash once available,
- required gates and artifact/review paths.

Treat Codex thread status as advisory, not authoritative. `idle` means "needs
inspection", not automatically "merged" or "done". `active` / `inProgress` also
needs inspection when the worktree state says otherwise. Read the recent thread
messages, check the worktree, and confirm whether a task commit or useful diff
exists before deciding to wait, merge, reject, or abandon.

## Stale And Stuck Session Handling

Do not let a delegated task block the orchestrator indefinitely just because the
Codex thread still reports `active` / `inProgress`. The orchestrator owns the
state machine and must reconcile thread status with git state.

Classify a delegated session as `stale-needs-inspection` when any of these are
true:

- The thread has been active for more than 15 minutes without a new final
  handoff, commit, status update, or meaningful worktree change.
- The worktree has a clean task commit but the thread is still `active` /
  `inProgress` or appears stuck before final handoff.
- The worktree has uncommitted changes but no recent progress, gate output, or
  explanation.
- Pending worktree setup has not resolved to a thread/worktree/branch within
  the expected setup window.

When stale is detected:

1. Inspect before acting:
   - `git status --short --branch`
   - `git log --oneline -3`
   - `git diff --name-status <default-branch>..HEAD`
   - `git diff --check <default-branch>..HEAD`
   - recent thread messages and any review/self-review document
2. If the worktree is clean and contains a task commit, treat it as
   `completed-unreviewed` even if the thread status is still `active`. Review
   the commit directly. If it passes, merge/push/cleanup/archive from the
   orchestrator and note that the final handoff was stuck.
3. If the worktree has useful uncommitted scoped changes, either send a targeted
   same-task nudge or take over the task in the orchestrator. Do not dispatch
   unrelated work while that diff is unresolved.
4. If the worktree has no useful diff/commit, record the stale condition and
   only remove the worktree or archive the thread after verifying the branch can
   be safely abandoned.
5. Notify the user on stale takeover, rejection, destructive cleanup, or a
   blocker. Quiet heartbeats are acceptable only when no action is needed.

This policy is a guardrail against silent overnight stalls. Heartbeats are a
watchdog, not proof that a child thread is making progress.

## Session And Worktree Setup

For implementation, proof, or documentation tasks that may create commits,
launch each delegated session in a separate Codex App project worktree by
default. Use the integration/local checkout only for quick read-only inspection,
orchestrator review, merge, push, and cleanup.

When Codex App worktree sessions are not available, replace "Codex App
worktree" in the task contract with the environment's supported isolated worker
mechanism. The invariant is isolation plus verifiable repo truth, not a specific
product surface.

Before dispatching, confirm that the repository is available as a saved Codex
App project when you intend to use Codex App worktree sessions. If project
thread creation fails with an unknown `projectId`, missing saved project, or
pending setup that never resolves, classify it as a setup blocker. Do not treat
that as an active worker.

The orchestrator should not implement a fresh delegated task just because
session/worktree dispatch failed. If a new task has not actually started, stay
in the orchestration layer: report the dispatch/tooling blocker, fix the
dispatch method, or ask for human input. Direct orchestrator implementation is
allowed only as a stale same-task takeover when there is already a scoped useful
diff/commit to rescue, or when the user explicitly asks the orchestrator to do
the task itself. When taking over, record why takeover was safer than waiting or
re-dispatching, then keep the write set to the original task contract.

If a fallback worker/subagent path is used after Codex App dispatch fails, the
fallback must still run in an isolated worktree or another explicitly isolated
checkout. Never let a fallback worker switch branches, edit files, or commit in
the orchestrator's integration/local checkout. If you cannot first create and
verify an isolated fallback checkout (`pwd`, branch, `git status --short
--branch`), stop and report the setup blocker instead of delegating.

Codex App worktree creation has one important API gotcha: a worktree
`startingState.branchName` is an existing starting ref, not the new task branch
to create. Do not pass a fresh desired task branch such as
`codex/<task-slug>` as `startingState.branchName` unless that ref already
exists. Let the App create the worktree from the saved project/current base, then
tell the delegated session to create or switch to `codex/<task-slug>` inside its
own worktree. If `create_thread` returns only a `pendingWorktreeId`, record it
as `pending setup` and poll repo/thread truth; do not assume the task is running,
and do not dispatch a duplicate same-task worker until setup resolves or is
declared stale/blocked.

Do not try to bind a newly hand-made git worktree path to `create_thread` as if
it were a saved Codex App project. Codex App project-thread creation requires a
saved `projectId`; arbitrary local paths are not accepted through that target.
If App worktree setup is unavailable or repeatedly pending, either report the
tooling blocker or use an explicitly supported worker/subagent path whose prompt
hard-requires the intended worktree, then immediately verify `pwd`, branch, and
`git status` before allowing edits.

For a new task, prefer a fresh Codex session with a compact task contract over
forking or delegating from a long orchestrator thread. Pass only the base commit,
allowed/forbidden paths, required source files, gates, evidence labels, and
handoff format. Long inherited context is a liability: it wastes context budget,
pulls in stale completed tasks, and can blur the current task boundary. Reuse an
existing delegated session only for the same task's rework, extra verification,
diff explanation, or review follow-up after orchestrator feedback.

When a phase changes, such as moving from many small evidence closures to a
larger feature module, retire the old long orchestrator instead of stretching it
indefinitely. Start a fresh orchestrator that reads the repository's current
source-of-truth docs (project rules, progress, roadmap, and recent reviews)
before dispatching new work. Treat repository docs and merged commits as the
handoff surface, not the compressed chat history.

## Anti-Shallow-Slice Gate

Do not let "do not reopen old first slices" become "rename the same shallow
slice and keep going." That rule exists to prevent repeated shallow closure, not
to justify more shallow work.

Before dispatching a new implementation/proof task in a domain that already has
a first closure, the orchestrator must classify the task as one of these:

- `vertical-completion`: connects already-landed pieces into a more complete
  end-to-end flow, such as UI action -> command/API -> persistence/projection ->
  readback/audit.
- `runtime-proof`: proves an existing local/proxy path in a real browser,
  device, LAN, hardware, or production-like runtime, with evidence labels kept
  honest.
- `blocked-removal`: removes a named blocker that prevents the next complete
  flow, such as a stale device path, missing write API, missing auth seam, or
  missing readback guard.
- `owner-gated`: records the exact human/product/accounting/payment/provider
  decision that blocks the next complete flow, without pretending it is
  implementation progress.

If a candidate is only another read-only shell, placeholder page, static review,
copy checklist, local fixture summary, or first guard in an already-partial
domain, reject or rewrite it unless it clearly removes a named blocker. The task
prompt must answer: what complete feature path does this advance, what previous
partial closure does it build on, and what will still remain after this slice.

For domains with several partial closures, prefer fewer larger vertical tasks
over many small horizontal tasks. It is acceptable for a vertical task to remain
local/proxy when hardware, live provider, or production is unavailable, but it
still must exercise a coherent local flow rather than just add another isolated
surface.

If the same domain has two or more merged partial closures and no single small
blocker is preventing progress, stop dispatching standalone slices and promote
the work to a feature-package plan. The package plan should name the user-visible
or operator-visible capability, list the minimum worker branches needed to make
it coherent, and define the merge order. A package may still use small worker
branches internally, but each branch must be tied to the package outcome and
must not exist merely to add another page, shell, fixture, or checklist.

Each delegated session should:

- start from the current accepted base commit or branch,
- create or switch to a `codex/<task-slug>` branch inside its own worktree,
- run `git status --short --branch` before editing,
- preserve unrelated dirty work if it encounters any,
- commit only its own scoped changes,
- leave merge, push, worktree removal, and branch deletion to the orchestrator unless the prompt explicitly says otherwise.

The orchestrator owns the lifecycle: create the worktree/session, record the thread/worktree/branch, review the finished branch, merge or reject it, then remove the worktree and delete the local branch. Whoever creates a worktree is responsible for cleaning it up after merge, rejection, or abandonment.

## Dispatch Prompt Contract

Each delegated session prompt should include:

- Task ID and plain-language outcome.
- Dependency/base commit or branch.
- Worktree and branch requirement.
- Allowed paths.
- Forbidden paths.
- Hardware/env ownership and mutual exclusions.
- Required source files/rules to read.
- Acceptance commands.
- Required docs/review/artifact updates.
- Evidence labels: `direct`, `proxy`, `blocked`.
- Anti-shallow-slice classification: `vertical-completion`,
  `runtime-proof`, `blocked-removal`, or `owner-gated`.
- Explanation of why this is not repeating an already-completed first slice.
- Requirement to self-review before handoff.
- Final handoff format: branch, commit, changed files, gates, evidence, risks.

Always include:

```text
Use a separate isolated worktree/session for this task unless the orchestrator explicitly says otherwise.
Start by running git status --short --branch. If you are not on the task branch, create or switch to codex/<task-slug>.
Do not start subagents, do not use another orchestrator, and do not create second-level delegation.
You are not alone in the codebase. Do not revert unrelated work; adapt to current changes.
If the run needs a human physical/device/payment/deploy action, proactively notify the user in their preferred language using the project's available notification mechanism; do not require the user to remember any skill name or command. Pause at the checkpoint, say the exact action, device/resource, what not to do, and what the user should reply; continue only after confirmation and record it in artifacts.
Before handoff, review your own diff as a reviewer: check boundaries, forbidden paths, shared contracts, docs drift, evidence strength, gates, anti-shallow-slice classification, and residual risks. Fix scoped issues you find before committing.
Commit to the task branch, but do not merge, push the integration branch, delete the worktree, or delete the branch unless the orchestrator explicitly asks you to.
```

Prefer prompts that close one feature path more fully over prompts that create
another shallow first slice. If a domain already has several partial closures,
bias the next task toward finishing one path end to end, unless a shared
contract or environment blocker must be resolved first. If no such task is
available, report the blocker instead of filling the queue with low-value
surface work.

## Feature-Package Planning Gate

Use this gate when the user asks for larger functional work, when a roadmap area
has accumulated several local/source/proxy closures, or when the orchestrator is
about to dispatch another task in the same domain.

Before dispatching, answer these questions in the orchestrator thread:

- What is the feature package or subproject milestone, in user/operator terms?
- Which existing closures does it build on?
- What is the smallest coherent end-to-end capability this package should
  deliver?
- Which work must be serial because it changes shared contracts, migrations,
  APIs, command/event handlers, or runtime ownership?
- Which work can run in parallel because the write sets and evidence surfaces
  are disjoint?
- What evidence will prove the package, and what remains `blocked`,
  `owner-gated`, or `runtime-proof` after the package lands?

Prefer package-sized outcomes such as:

- UI action -> API/client -> persistence/projection -> readback/audit.
- Admin/Manager operational flow across list/detail/write/readback states.
- Local runtime proof for a previously source-only flow.
- Hardware/production/provider checkpoint only after code paths and rollback evidence
  are ready.

Do not use package planning as permission for a huge unreviewable branch. The
orchestrator should still split implementation into mergeable worker contracts,
but the contracts should form a visible milestone rather than a queue of tiny
unrelated improvements.

## Dynamic Heartbeat Prompt Pattern

Do not hard-code old task IDs into a long-lived automation. Use dynamic discovery:

```text
Check the orchestrator thread, recent delegated sessions, pending worktree setup, git worktree list, and integration-branch repo status. Identify tasks created by this thread that are active, pending, completed but unmerged, blocked, or stale.

If a task completed with a commit, validate diff, self-review, boundaries, docs/reviews/artifacts, and gates. If it passes, merge to the default integration branch, push when normal for this project or explicitly requested, remove the task worktree, and delete the local branch. If it fails, report blocking findings and do not merge.

If tasks are still running, reconcile thread status with git state. A thread
that is `active` but has a clean task commit is not "still running" for
orchestration purposes; review the commit directly. A thread that is `active`
with no progress beyond the stale threshold is `stale-needs-inspection`, not a
reason to wait forever. Do not edit code or touch shared hardware unless taking
over a stale same-task worktree under the stale-session policy. Notify only on
material progress, stale takeover, blocker, completion, cleanup, or conflict.

If no active/pending/unreviewed tasks remain and the default branch is clean, choose the next batch from the current roadmap. Default max concurrency is 2; allow 3 only after shared contracts are merged, no hardware/production/payment task is active, and all write sets are plainly disjoint. Serialize shared contracts and hardware. Require self-review in every new prompt.
```

Keep task IDs in the conversation/session records, not in the persistent automation prompt. If a temporary watchlist is useful, treat it as disposable and update it immediately after completion.

If the task set changes materially, update the automation prompt. If the heartbeat is obsolete, delete it instead of letting it keep waking the thread with stale instructions. Prefer the Codex App automation tool over hand-written recurring prompt text when creating, updating, viewing, or deleting automations.

When the user cancels an automation or starts work manually again, report how to
restart the same orchestration mode naturally: open a fresh orchestrator session,
ask it to read the current repo docs, then let it choose and dispatch bounded
tasks from the current roadmap. The user should not need to remember internal
task IDs or skill names.

## Concurrency Rules

Default to two sessions. Allow up to three only for low-risk post-contract
fan-out where the default branch is clean and the write sets are plainly
disjoint. Use one when:

- a shared contract is being edited,
- hardware/production/payment ownership is involved,
- the next step depends on a result from the current task,
- the repo has unresolved dirty or merged-but-unpushed state,
- the task is likely to create migration/proto/API conflicts.

Use two when write sets are disjoint, for example:

- hardware proof + docs/runbook task,
- frontend read-only surface + backend docs audit,
- Terminal UI + Cloud read model after contract is already merged,
- implementation + independent verification/docs drift pass.

Use three only when all of these are true:

- no shared contract, migration, Cloud API, hardware/production/payment, or long-running proof is active,
- the base contract branch has already been accepted and merged to the default
  integration branch,
- the default branch is clean and pushed or intentionally local-only for this
  project,
- each task has a separate module/write set and separate review/artifact path,
- the orchestrator can realistically review and merge the resulting branches
  without batching unresolved conflicts.

Do not open more sessions just because idle capacity exists. Parallelism should reduce calendar time without increasing merge risk.

Before dispatching more work, check whether existing active sessions have
uncommitted changes or occupy hardware. Do not start a new task just because the
integration checkout is clean if an active worktree is still producing evidence
or holding a device/resource.

For proto/envelope tasks, "serial" means no parallel consumers until the
contract branch is accepted, not necessarily "contract files only." If the
acceptance gate enforces consumer synchronization, the serial task must either
include the smallest required consumer updates or explicitly report that the
contract cannot be merged under current boundaries.

## Evidence Discipline

For hardware, payment, deploy, and environment work:

- Record owner, start/end time, device/env facts, install/clear-data/reboot/config changes.
- Label proof as `direct`, `proxy`, `local`, or `blocked`.
- Do not upgrade `SENT`, local unit tests, TCP reachability, screenshots, or proxy services into direct proof.
- If human action is needed, pause at a safe checkpoint and use the available
  notification/voice mechanism for the environment. The prompt must state the
  exact action, device/resource, what not to do, and what the user should reply.
  Continue only after user confirmation, and record the prompt, confirmation,
  and observed device/env evidence in the artifact. If no reliable notifier is
  available, report that limitation before starting the human-dependent proof.
- Avoid starting a session that cannot finish defensibly because the required human action, payment backend, physical device, or deploy owner is unavailable.
- Do not track raw runtime dumps, secrets, credentials, unnecessary database snapshots, or oversized logs. Keep artifacts minimal, redacted, and useful for review.
- When re-running gates during orchestrator review, use a temporary artifact directory where possible so verification does not dirty the submitted proof artifacts. If a stronger post-merge gate passes because the review environment has extra dependencies, report it as additional evidence without rewriting the original session's claimed evidence.

## Review And Merge Checklist

For every completed session:

```text
git status --short --branch
git log --oneline -3
git diff --name-status <default-branch>..HEAD
git diff --check <default-branch>..HEAD
```

Then inspect:

- changed files match the prompt boundary,
- no forbidden shared contracts changed,
- self-review exists in final message or review doc,
- docs/progress/review/artifacts match the actual evidence,
- generated artifacts are useful and not raw runtime dumps,
- acceptance gates are appropriate and not just decorative.

Also check:

- the default branch is clean before merging,
- the task worktree is clean after its commit,
- `git diff --name-status <default-branch>..HEAD` does not include another
  task's files,
- the final message or review doc includes a real self-review,
- progress/roadmap updates are factual and do not mark partial work as done,
- direct/proxy/local/blocked labels match the artifacts.

If clean:

```text
git merge --no-ff <task-branch> -m "merge: <scope>"
<run post-merge gates>
git push <remote> <default-branch>  # when normal for the project or requested
git worktree remove <task-worktree>
git branch -d <task-branch>
```

Resolve simple progress-doc conflicts by preserving both entries. Do not rewrite unrelated user work.

If the conflict is in a shared contract, migration, core aggregate, protocol envelope, or evidence artifact, stop and review manually instead of forcing a merge.

## When To Stop Dispatching

Stop or ask for human input when:

- all remaining tasks need hardware, payment backend, production credentials, or production access,
- a shared contract must be decided before more consumers can proceed,
- current tasks are running and new work would compete for the same files/resources,
- the roadmap is stale enough that choosing work would be speculation,
- multiple next tasks require product priority decisions.

When stopping, report the active sessions, clean repo state, next best tasks, and blockers.
