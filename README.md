[English](README.md) | [中文](README.zh-CN.md)

# codex-orchestrator

**A Codex App-first harness for Loop Engineering.** `codex-orchestrator`
turns Codex App into a supervised engineering loop: splitting roadmap work into
isolated worktree sessions, checking them with heartbeats and local policy
guards, reviewing and merging clean branches, rescuing stuck sessions, and
dispatching the next batch when it is safe to do so.

## 🚀 Quick Start

Open Codex App in the repository you want to orchestrate and paste:

```text
I want to try codex-orchestrator in this repository.

Read https://github.com/indiekitai/codex-orchestrator and use it as a
Codex App-first orchestration workflow.

If the Codex App skill from that repository is not installed, install it into
~/.codex/skills/codex-orchestrator.

If the Go helper CLI is useful for durable ledger state, explain what it does
and then install or build it if safe.

Start with a dry run:
- inspect git status, worktrees, and project docs;
- explain how you would split work into isolated Codex worktree sessions;
- explain what you would monitor, review, merge, push, and clean up;
- label evidence as direct, proxy, local, or blocked.

Do not push, deploy, delete worktrees, or make destructive changes unless I
explicitly approve.
```

Codex should read this repository, install the Codex App skill if it is missing,
decide whether the helper is useful for the current project, and produce a
dry-run orchestration plan before doing mutating work.

When durable state is useful, Codex can use the `codex-orchestrator` helper
binary for a local ledger, `observe`, heartbeat reports, and routine checks.
Users do not install this through Homebrew, npm, or another package-manager
route. The product route is: give the GitHub repository to Codex App, then let
Codex install/read/use the skill and helper only if needed.

If you are evaluating the workflow for the first time, use this order:

1. Paste the prompt into Codex App from the repository you want to orchestrate.
2. Let Codex read this GitHub repository and install or update the skill if needed.
3. Ask for a read-only dry run, and do not create workers or sessions yet.
4. Wait for explicit user approval after the dry-run plan before any merge,
   push, cleanup, or worker creation.
5. Treat the Go helper as optional support for ledger state, heartbeat reports,
   and routines, not something you must learn before the trial.
6. If you want a real-project example first, read
   [docs/case-studies/tastyfuture-orchestration.md](docs/case-studies/tastyfuture-orchestration.md).

Naming note: **codex-orchestrator** is the product name, repository name,
Codex App skill name, and helper CLI name.

## 🔥 The Problem

Running one Codex session at a time is fine for small tasks. But for anything larger — a new API with 4 endpoints, a module rewrite, a multi-service feature — you hit real pain:

- **Context switching**: Manually checking "is session 3 done yet?" while session 1 needs a merge
- **Stuck sessions**: A session hangs at 80% complete. You don't notice for an hour
- **Merge conflicts**: Two sessions edit the same proto file. Both finish. Neither merges cleanly
- **Overnight babysitting**: You want to dispatch 3 tasks before bed but can't trust them unsupervised

## 🏗️ How It Works

```
                    ┌─────────────────────┐
                    │   Orchestrator      │
                    │   (main thread)     │
                    └──────┬──────────────┘
                           │
              ┌────────────┼────────────────┐
              ▼            ▼                ▼
     ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
     │  Session A   │ │  Session B   │ │  Session C   │
     │  worktree/a  │ │  worktree/b  │ │  worktree/c  │
     │  branch: a   │ │  branch: b   │ │  branch: c   │
     └──────┬───────┘ └──────┬───────┘ └──────┬───────┘
            │                │                │
            ▼                ▼                ▼
     ┌──────────────────────────────────────────────┐
     │            5-min heartbeat loop              │
     │  ┌─ check git state ──────────────────────┐  │
     │  │  committed? → review → merge → cleanup │  │
     │  │  stuck + commit? → review → merge       │  │
     │  │  stuck + diff?   → nudge to continue   │  │
     │  │  active?    → let it cook              │  │
     │  └────────────────────────────────────────┘  │
     │  All done? → dispatch next batch             │
     └──────────────────────────────────────────────┘
```

## ✨ Key Features

| Feature | What it does |
|---------|-------------|
| **Bounded task contracts** | Each session gets a precise scope: allowed paths, forbidden paths, base commit, acceptance gates, evidence labels |
| **Automatic concurrency control** | Default 2 sessions, up to 3 when write sets are disjoint. Serializes shared contracts (protos, migrations, APIs) |
| **5-minute heartbeat** | Periodic check reconciles thread status with actual git state — no silent overnight stalls |
| **Stuck session recovery** | If a session is idle >15 min: has a clean commit → review and merge directly; has uncommitted useful changes → send a targeted nudge to continue; no useful diff → mark abandoned |
| **Anti-shallow-slice gate** | Rejects "another placeholder page" tasks. Forces vertical completion, runtime proof, or blocker removal |
| **Evidence discipline** | Labels proof as `direct`, `proxy`, `local`, or `blocked`. No upgrading unit tests into production proof |
| **Self-review enforcement** | Every session must review its own diff before handoff. The orchestrator re-reviews before merging |
| **Feature-package planning** | When a domain has multiple partial closures, promotes work to a coherent milestone instead of more tiny slices |
| **Continuous operation** | Doesn't stop after one feature — reads roadmap, picks the next buildable feature, dispatches, repeats. Designed for overnight/unattended multi-feature runs |
| **Continuation guard** | A task-specific heartbeat can stop only after the orchestrator checks whether the broader queue should continue |

## ✅ Prerequisites And Safety

This repository is a Codex skill/runbook, not a standalone background daemon.
The fully autonomous loop depends on the host environment exposing compatible
capabilities, especially:

- creating or continuing isolated Codex sessions,
- creating separate git worktrees or equivalent isolated worker environments,
- checking thread status and worktree git state,
- creating/updating recurring automations or heartbeat reminders,
- merging and pushing through normal project git policy.

If those tools are unavailable, the skill should degrade into a manual
orchestration checklist: dispatch fewer sessions, inspect git state directly,
and stop before pretending that monitoring, merge, push, or cleanup happened.

For open-source use, start with a dry run on a disposable repository or feature
branch. Keep automatic push disabled until you trust the review gates and your
project's branch protection policy.

The core skill does not require Python. The v2 helper is a Go CLI that can be
built as a single binary. The Python helper remains as a prototype and
compatibility reference.

## 🚫 What This Is Not

This is not a replacement for engineering judgment, code review, or production
verification. It is a way to make AI-assisted development more structured:
bounded tasks, isolated worktrees, explicit evidence labels, and review before
merge.

The goal is not to let agents write unattended forever. The goal is to keep the
human in the loop at the right leverage point: designing the loop, reviewing the
evidence, and deciding what should ship.

It is also not trying to be a full agent operating system. That route is out of
scope for this project. The practical target is narrower: a reliable harness
around Codex App sessions, with durable state, recovery rules, verification
routines, policy/eval checks, and honest evidence labels.

## 🚀 Codex App Setup Flow

If you want Codex App to do the setup for you, use the bootstrap prompt above.
Codex should inspect this repository and then perform the relevant steps:

```bash
# Install the Codex App skill when needed.
cp -r codex-orchestrator ~/.codex/skills/codex-orchestrator

# Optionally install the helper when durable state is useful.
scripts/install.sh
codex-orchestrator init
```

You can also download a prebuilt `codex-orchestrator_<os>_<arch>` binary from
the Releases page and put it on your `PATH`, but that is an advanced/helper
path, not a package-manager distribution channel. Most users should start by
asking Codex App to read this repository.

The intended setup order stays the same even when the helper is used:

1. Give Codex App the GitHub repository.
2. Let Codex read `README.md`, `SKILL.md`, and the setup docs.
3. Let Codex decide whether skill install or helper build is useful.
4. Require a read-only dry run and wait for explicit user approval before any
   worker creation or mutating orchestration step.

For release assets and shell completions, see
[docs/distribution-package.md](docs/distribution-package.md).

After setup, ask Codex App to use codex-orchestrator. Codex may invoke the
installed skill when appropriate:

```
Use codex-orchestrator to split this feature into bounded worktree sessions,
review/merge completed branches, and dispatch the next batch.
```

Or be specific:

```
I need to build a REST API with user auth, CRUD endpoints,
pagination, and rate limiting. Use codex-orchestrator to run this as parallel
sessions overnight.
```

The orchestrator will:
1. Decompose the work into bounded task contracts
2. Dispatch sessions into separate worktrees
3. Run a heartbeat loop every 5 minutes
4. Review and merge completed sessions
5. Rescue stuck sessions by harvesting their commits
6. Dispatch the next batch when slots open up

With the v2 helper installed, it can also persist task state in
`.codex-orchestrator/ledger.json` and write heartbeat reports that a fresh
orchestrator session can resume from.

If this is your first trial, ask Codex App to follow the safer
disposable-repository path in
[docs/beta-usability-package.md](docs/beta-usability-package.md) before running
the workflow on a real project.

## 📋 Real Example

**Goal**: Build a REST API with 4 major components.

The orchestrator decomposes it into parallel sessions:

```
Session A: codex/api-auth
  Allowed: src/auth/**, src/middleware/auth.ts, tests/auth/**
  Forbidden: src/db/migrations/**, src/api/products/**
  Gate: npm test -- --grep auth

Session B: codex/api-products
  Allowed: src/api/products/**, src/models/product.ts, tests/products/**
  Forbidden: src/auth/**, src/db/migrations/**
  Gate: npm test -- --grep products
```

Sessions A and B run in parallel (disjoint write sets). After both merge, the orchestrator dispatches:

```
Session C: codex/api-pagination
  Allowed: src/middleware/pagination.ts, src/api/**/router.ts, tests/pagination/**
  Gate: npm test -- --grep pagination

Session D: codex/api-rate-limit
  Allowed: src/middleware/rateLimit.ts, src/config/limits.ts, tests/rateLimit/**
  Gate: npm test -- --grep rateLimit
```

Overnight, the heartbeat catches Session C stuck at minute 22 with a clean commit. The orchestrator reviews the commit directly, merges it, and moves on — no human intervention needed.

## 🪜 Loop Engineering Maturity Model

`codex-orchestrator` is a practical **Codex App-first harness**, not the final
form of agentic software development. It sits between manual prompting and a
future persistent agent runtime.

Worker sessions still own the inner edit/test/fix loop. This project manages
the outer engineering loop around them: task selection, isolation, monitoring,
review, merge, cleanup, and continuation.

| Level | Shape | What changes |
|-------|-------|--------------|
| **v0: Prompting** | Human prompts one agent at a time | The human is the scheduler, reviewer, and recovery loop |
| **v1: Supervised orchestrator skill** | `codex-orchestrator` today | Worktree isolation, bounded task contracts, heartbeat monitoring, review/merge discipline, evidence labels |
| **v2: Persistent task ledger** | A real state store behind the loop | Tasks, attempts, worker state, gates, blockers, and outcomes survive across threads and restarts |
| **v2.5: Verification routine foundation** | Routine contracts become inspectable | Shared output schema, evidence labels, harness map, and validator for reusable routines |
| **v3: Routine library** | Reusable background routines | PR reviewer, CI fixer, stale-session rescuer, rebase helper, docs drift checker, release verifier |
| **v4: Eval and safety layer** | Failures become tests and policies | Orchestration policy auditor, prompt-injection cases, dangerous-operation classifiers, permission checks, evidence-quality evals |

This repository intentionally starts at v1 because that is the layer most teams
can adopt today without running a custom daemon or changing their whole
development platform. The next hard problems are recovery classification,
runtime verification, policy/eval coverage, and reviewable rule improvement.

An agent operating system is deliberately not on this roadmap. The project
should stay focused on making Codex App orchestration more observable,
recoverable, and reviewable.

The ambition is not to claim that a Codex skill is already a complete Loop
Engineering runtime. The ambition is to make the first useful outer loop
concrete: bounded work, isolated execution, heartbeat inspection, honest proof
labels, and review before merge.

See [docs/v2-persistent-ledger-and-heartbeat.md](docs/v2-persistent-ledger-and-heartbeat.md)
for the v2 durable ledger and heartbeat helper design, and
[docs/v2-usage.md](docs/v2-usage.md) for the Codex App + Go helper workflow.
See [docs/routines/README.md](docs/routines/README.md) for the v2.5 routine
contract format and [docs/routines/harness-map.md](docs/routines/harness-map.md)
for the feedback-loop harness model.
For a first-time external-user path from install to a safe local demo, see
[docs/beta-usability-package.md](docs/beta-usability-package.md). For release
copy, see [docs/beta-release-notes-draft.md](docs/beta-release-notes-draft.md).
For a research note on how this maps to Loop Engineering, see
[docs/research/loop-engineering-alignment.md](docs/research/loop-engineering-alignment.md).
For the harness reading notes that de-scope the agent-OS route, see
[docs/research/harness-reading-notes.md](docs/research/harness-reading-notes.md).
For the broader roadmap, see [docs/roadmap.md](docs/roadmap.md).

The v2 helper CLI currently supports:

```bash
go build -o codex-orchestrator ./cmd/codex-orchestrator
./codex-orchestrator init
./codex-orchestrator record-task --id TASK --worktree /path/to/wt --branch codex/task --max-runtime-minutes 90 --review-budget-minutes 25
./codex-orchestrator observe
./codex-orchestrator heartbeat --count 1 --write-report .codex-orchestrator/heartbeat-report.json
./codex-orchestrator status
./codex-orchestrator append-event --type review --task-id TASK --status completed-unreviewed
./codex-orchestrator validate-routines --dir routines
./codex-orchestrator run-routine pr-reviewer --task-id TASK --write-report /tmp/pr-reviewer-report.json
./codex-orchestrator run-routine stale-task-rescuer --task-id TASK --write-report /tmp/stale-task-rescuer-report.json
./codex-orchestrator run-routine ci-fixer --task-id TASK --write-report /tmp/ci-fixer-report.json
./codex-orchestrator run-routine release-verifier --tag v0.3.0-alpha.1 --write-report /tmp/release-verifier-report.json
./codex-orchestrator run-routine docs-drift-checker --write-report /tmp/docs-drift-checker-report.json
./codex-orchestrator run-routine evidence-label-auditor --write-report /tmp/evidence-label-auditor-report.json
./codex-orchestrator run-routine orchestration-policy-auditor --write-report /tmp/orchestration-policy-auditor-report.json
./codex-orchestrator run-routine roadmap-next-task-suggester --write-report /tmp/roadmap-next-task-suggester-report.json
./codex-orchestrator run-routine budget-policy-report --write-report /tmp/budget-policy-report.json
./codex-orchestrator policy check --write-report /tmp/policy-check-report.json
./codex-orchestrator eval run --write-report /tmp/eval-run-report.json
./codex-orchestrator eval add-failure --id dry-run-example --text "Dry run mode can dispatch workers immediately." --expect OPA001=1
./codex-orchestrator rules propose --from-review docs/reviews/example.md --write-report /tmp/rules-proposal-report.json
./codex-orchestrator record-routine-run --routine pr-reviewer --status passed --evidence-local "go test ./..." --action "reviewed diff" --next "merge branch"
./codex-orchestrator record-routine-run --report-json examples/routine-reports/pr-reviewer.passed.json
```

The JSON heartbeat report includes `overallStatus`, per-status `counts`, a
`reviewPressure` block, read-only `budgetSummary`, and additive
`budgetPressure` warnings. Per-task runtime/review budgets recorded with
`record-task` are surfaced in `observe`, `status`, and heartbeat summaries for
visibility only. Runtime pressure is computed from local ledger timestamps;
review pressure is computed only when a review-ready timestamp is recorded.
Missing or indeterminate budget data is labeled as local/static helper evidence.
The helper does not kill processes, schedule sessions, or enforce budgets.

Codex App worktree dispatch is App-first. Save the repository as a Codex App
project before relying on project worktree sessions. If dispatch fails because
the project is unknown, or setup never resolves to a real worktree/thread,
treat it as a setup blocker. Do not let a fallback worker edit the
orchestrator's own checkout; first create and verify an isolated fallback
worktree, or stop and report the blocker.

`run-routine pr-reviewer` is the first runnable routine MVP. It is read-only
against the task worktree: it loads the ledger task, checks worktree and branch
state, records `git status --short --branch`, compares `baseCommit..HEAD`,
captures `git diff --name-status`, and runs `git diff --check`. It writes a
standard `RoutineRunReport` JSON that can later be recorded with
`record-routine-run --report-json`. It does not merge, push, delete branches,
clean worktrees, run task-specific test gates, or claim runtime proof.

`run-routine stale-task-rescuer` is the second runnable routine MVP. It is also
read-only against the task worktree: it loads the ledger task by id, records
ledger status, last observation, and recent task history, verifies worktree and
branch state, captures `git status --short --branch` and `git log --oneline -3`,
then classifies rescue readiness from local git state. A clean task with
commits after `baseCommit` passes with the next action set to orchestrator
review of the committed diff. Useful uncommitted changes fail with evidence and
a same-worker or same-task takeover recommendation. Missing worktrees, branch
mismatches, missing `baseCommit`, or git inspection failures block. The runner
does not modify ledger status, stage, commit, merge, clean worktrees, dispatch
new work, or claim direct/proxy runtime proof; MVP evidence is `local` or
`blocked` only.

`run-routine ci-fixer` is the third runnable routine MVP. Despite the name, it
does not edit code or auto-fix CI. It does execute trusted gate commands already
recorded on the ledger task, so do not run it against an untrusted repository or
untrusted ledger. It loads the ledger task by id, verifies the task worktree and
expected branch, refuses dirty worktrees, compares `baseCommit..HEAD`, records
the committed file list, and runs those recorded gates in the task worktree
with a local timeout. Passing gates plus committed work after `baseCommit`
return `passed` with a next action to run the orchestrator review/merge flow.
Dirty worktrees or failing gates return `failed` and send the task back to the
same worker or a same-task takeover. Missing gates, missing `baseCommit`, branch
mismatches, or git inspection failures return `blocked`. It does not stage,
commit, merge, push, clean worktrees, modify ledger status, or claim
direct/proxy runtime proof; MVP evidence is `local` or `blocked` only.

`run-routine release-verifier` is the fourth runnable routine MVP. It is
read-only and does not load or update the ledger. It verifies a supplied local
git tag, reads GitHub release metadata through `gh release view` when `gh` is
available, checks alpha/beta/rc prerelease flags, and compares release asset
names against this repo's default Go CLI asset set or repeated
`--expected-asset` overrides. Missing tags, missing releases, drafts,
prerelease mismatches, and missing assets return `failed`; unavailable `gh`,
auth/network failures, or unparseable release metadata return `blocked`. It
does not create or edit releases, move tags, upload assets, stage, commit,
merge, push, clean, dispatch, mutate the ledger, or claim production/runtime
proof; MVP evidence is `local`, `proxy`, or `blocked`.

`run-routine docs-drift-checker` is the fifth runnable routine MVP. It is
read-only and does not load or update the ledger. It parses the local
`run-routine` command surface from `cmd/codex-orchestrator/main.go`, compares
the runnable routine IDs with `routines/*.json`, and scans `README.md`,
`README.zh-CN.md`, `SKILL.md`, `docs/routines/README.md`, `docs/v2-usage.md`,
and `docs/roadmap.md` when present for obvious missing routine references or
stale status text. Missing docs references or missing specs return `failed`; missing
repository/source/spec access returns `blocked`. It does not stage, commit,
merge, push, tag, release, clean worktrees, dispatch sessions, mutate the
ledger, or claim runtime proof; MVP evidence is `local` or `blocked`.

`run-routine evidence-label-auditor` is the sixth runnable routine MVP. It is
read-only and does not load or update the ledger. It scans explicit repo-local
docs, routine specs, routine report JSON files, and ledger-shaped JSON for
obvious evidence-label issues: weak evidence wording near strong proof wording,
RoutineRunReport JSON missing the `direct` / `proxy` / `local` / `blocked`
buckets, and direct evidence recorded for routines whose specs explicitly
reserve direct evidence. It applies deterministic named policy/eval rules
(`ELA001`-`ELA009`), treats glossary/prohibition/blocked-definition wording as
allowed negatives, and reports local rule-hit summaries when findings are
present. Findings are heuristic suspicions, not proof of wrongdoing. It does
not stage, commit, merge, push, tag, release, clean worktrees, dispatch
sessions, mutate the ledger, or claim runtime proof; MVP evidence is `local`
or `blocked`.

`run-routine orchestration-policy-auditor` is the first V4 policy/eval routine
MVP. It is read-only and does not load or update the ledger. It scans
repo-local orchestration docs, prompts, routine specs, routine reports, and
ledger/event files for deterministic orchestration policy rules (`OPA001`-
`OPA008`): dry-run dispatch barrier, no-main-checkout fallback guard, heartbeat
continuation guard, delegated worker boundaries, evidence promotion boundaries,
heartbeat target binding guard, pending worktree ledger guard, and budget-policy
evidence/control boundary drift. Findings are
local/static suspicions, not proof of wrongdoing. It
does not stage, commit, merge, push, tag, release, clean worktrees, dispatch
sessions, mutate the ledger, or claim runtime proof; MVP evidence is `local`
or `blocked`.

`policy check` is the first product-facing V4 policy/eval command. It wraps the
read-only orchestration policy auditor and also runs transcript-backed local
eval fixtures from `eval/orchestration-policy-auditor/`. The initial fixtures
cover the failures this project already encountered in real orchestration:
dry-run dispatch without explicit approval, main-checkout fallback after
worktree setup failure, stopping the larger queue after one child task,
delegated worker prompts missing core boundaries, local/proxy evidence
promotion, heartbeat automation bound to the literal `current` placeholder,
pending worktree ids kept only in prompt/chat state, and budget-policy helper
control or evidence overclaims. It does not dispatch Codex
sessions, mutate git, update the ledger, or claim runtime proof; the result is
local/static policy evidence.

`eval run` runs the policy fixture suite by itself. Use it when changing
policy rules and you want deterministic regression coverage without scanning
the current repository text. The first suite is
`orchestration-policy-auditor`; it reads fixtures from
`eval/orchestration-policy-auditor/` and compares actual `OPAxxx` hit counts
against each fixture's `expectedRuleHits`.

`eval add-failure` adds a manually supplied failure case to the fixture suite.
For the MVP, pass the text and expected rule hits explicitly. The command
verifies the text against the current policy rules before writing JSON, refuses
to overwrite an existing fixture unless `--force` is supplied, and does not
parse review documents automatically yet.

`rules propose` turns local evidence text or a review file into a review-only
rule proposal report. It can read `--from-review`, `--text`, or `--text-file`,
and it writes only the proposal report when `--write-report` is supplied. It
does not edit `SKILL.md`, README files, AGENTS/CLAUDE instructions, policy
files, or project rules; every proposal is marked as needing human review.

`run-routine roadmap-next-task-suggester` is the eighth runnable routine MVP.
It is read-only and does not mutate the ledger. It parses remaining candidate
tasks from `docs/roadmap.md`, compares them against local runnable routine IDs
and `routines/*.json`, optionally filters duplicate active or merged matches
from a repo-local `.codex-orchestrator/ledger.json`, and prefers conservative
read-only local tasks over mutating, release-scoped, or network-dependent
work. If only unsafe items remain, it returns a queue-drained next action
instead of pretending to dispatch. It does not stage, commit, merge, push,
tag, release, clean worktrees, dispatch sessions, mutate the ledger, or claim
runtime proof; MVP evidence is `local` or `blocked`.

`run-routine budget-policy-report` is a read-only local/static budget visibility
runner. It inspects roadmap/routine docs, routine budget metadata, optional
repo-local ledger state, and an optional heartbeat report when present. It
keeps budget metadata and heartbeat `budgetPressure` warnings as `local`
evidence, records unavailable live runtime/review timing as `blocked`, and
does not schedule, prioritize, pause, kill, dispatch, merge, push, delete,
clean worktrees, mutate the ledger, or enforce budgets.

When a delegated task is merged, pushed, released, and cleaned, the
task-specific heartbeat is not automatically the end of the loop. Before
deleting that heartbeat, the orchestrator should inspect ledger/repo truth and
the roadmap queue. If safe work remains, it should dispatch the next bounded
task or replace the heartbeat with a next-task monitor. Delete the heartbeat
only after the queue is drained, or after the next action is blocked and
reported.

## 🧱 Architecture

The orchestrator operates as a **state machine** over delegated sessions:

```
dispatch → active → completed-unreviewed → merged
                 ↘ stale-needs-inspection → rescued/abandoned
                 ↘ blocked → waiting for human input
```

**Key components:**

- **State Ledger**: Tracks task ID, thread ID, worktree, branch, base commit, write set, status, and gates for every session
- **Heartbeat Loop**: Every 5 minutes, reconciles Codex thread status with actual git state
- **Review Pipeline**: Diff boundary check, self-review verification, contract conflict detection, evidence label validation
- **Anti-Shallow-Slice Gate**: Classifies every task as `vertical-completion`, `runtime-proof`, `blocked-removal`, or `owner-gated`

## ⚖️ vs Manual Orchestration

| | Manual | codex-orchestrator |
|---|--------|-------------------|
| **Session monitoring** | You check each session tab manually | 5-min heartbeat auto-reconciles |
| **Stuck sessions** | You notice (eventually) and intervene | Auto-detected at 15 min, commit harvested |
| **Merge conflicts** | Discovered at merge time | Prevented by disjoint write-set enforcement |
| **Shallow work** | Sessions produce placeholder pages | Anti-shallow-slice gate rejects or rewrites |
| **Evidence honesty** | Trust the session's self-report | `direct`/`proxy`/`local`/`blocked` labels enforced |
| **Overnight runs** | You wake up to a mess | You wake up to merged branches |
| **Concurrency** | YOLO parallelism | Serialized contracts, max 2-3 with rules |

## ⚙️ Configuration

These parameters are tunable in the skill or per-dispatch:

| Parameter | Default | Description |
|-----------|---------|-------------|
| Max concurrency | 2 | Active sessions. Raise to 3 only when write sets are disjoint and no shared contracts are active |
| Stale threshold | 15 min | Time without progress before a session is flagged for inspection |
| Heartbeat interval | 5 min | How often the orchestrator checks all sessions |
| Branch prefix | `codex/` | Namespace for task branches |
| Push policy | Project-specific | Push only when normal for the repository or explicitly requested |
| Evidence labels | `direct`, `proxy`, `local`, `blocked` | Required classification for local, hardware, deploy, or payment proof |
| Anti-shallow-slice | Enforced | Tasks must be classified before dispatch |

## 📂 File Structure

```
codex-orchestrator/
├── SKILL.md              # The orchestrator skill (copy to ~/.codex/skills/)
├── agents/
│   └── openai.yaml       # Agent interface definition
├── .github/workflows/
│   └── release.yml       # Cross-platform release binary workflow
├── cmd/
│   └── codex-orchestrator/
│       ├── main.go       # Go helper CLI
│       └── main_test.go  # CLI state-machine tests
├── docs/
│   ├── beta-release-notes-draft.md
│   ├── beta-usability-package.md
│   ├── case-studies/
│   │   └── tastyfuture-orchestration.md
│   ├── distribution-package.md
│   ├── roadmap.md
│   ├── research/
│   │   └── loop-engineering-alignment.md
│   ├── reviews/
│   ├── routines/
│   │   ├── README.md
│   │   └── harness-map.md
│   ├── v2-usage.md
│   └── v2-persistent-ledger-and-heartbeat.md
├── routines/
│   ├── api-proof.json
│   ├── browser-runtime-proof.json
│   ├── ci-fixer.json
│   ├── database-proof.json
│   ├── device-proof.json
│   ├── docs-drift-checker.json
│   ├── evidence-label-auditor.json
│   ├── log-proof.json
│   ├── orchestration-policy-auditor.json
│   ├── pr-reviewer.json
│   ├── release-verifier.json
│   ├── roadmap-next-task-suggester.json
│   ├── budget-policy-report.json
│   └── stale-task-rescuer.json
├── examples/
│   ├── ledger.example.json
│   └── routine-reports/
│       ├── api-proof.blocked.json
│       ├── budget-policy-report.review-only.json
│       └── pr-reviewer.passed.json
├── scripts/
│   ├── build-release-assets.sh
│   ├── install.sh
│   ├── ledger_heartbeat.py
│   └── publish-release.sh
├── go.mod
├── README.md             # This file
├── README.zh-CN.md       # Chinese README
└── LICENSE               # MIT
```

## 📄 License

MIT

---

Built by [IndieKit.ai](https://indiekit.ai) — open-source developer tools for the AI-native workflow.
