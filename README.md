[English](README.md) | [中文](README.zh-CN.md)

# codex-orchestrator

**Loop engineering for OpenAI Codex App.** A multi-session orchestrator skill
that turns a single coding assistant into a supervised engineering loop:
splitting roadmap work into isolated worktree sessions, checking them with
heartbeats, reviewing and merging clean branches, rescuing stuck sessions, and
dispatching the next batch when it is safe to do so.

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
| **Evidence discipline** | Labels proof as `direct`, `proxy`, or `blocked`. No upgrading unit tests into production proof |
| **Self-review enforcement** | Every session must review its own diff before handoff. The orchestrator re-reviews before merging |
| **Feature-package planning** | When a domain has multiple partial closures, promotes work to a coherent milestone instead of more tiny slices |
| **Continuous operation** | Doesn't stop after one feature — reads roadmap, picks the next buildable feature, dispatches, repeats. Designed for overnight/unattended multi-feature runs |

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

The core skill does not require Python. The v2 helper now has a Go CLI seed that
can be built as a single binary. The Python helper remains as a prototype and
compatibility reference.

## 🚫 What This Is Not

This is not a replacement for engineering judgment, code review, or production
verification. It is a way to make AI-assisted development more structured:
bounded tasks, isolated worktrees, explicit evidence labels, and review before
merge.

The goal is not to let agents write unattended forever. The goal is to keep the
human in the loop at the right leverage point: designing the loop, reviewing the
evidence, and deciding what should ship.

## 🚀 Quick Start

### 1. Install the skill

```bash
# Copy to your Codex skills directory
cp -r codex-orchestrator ~/.codex/skills/delegated-session-orchestrator
```

### 2. Use it in Codex

Open a Codex session and tell it to orchestrate:

```
Use $delegated-session-orchestrator to split this feature into
bounded worktree sessions, review/merge completed branches,
and dispatch the next batch.
```

Or be specific:

```
I need to build a REST API with user auth, CRUD endpoints,
pagination, and rate limiting. Use $delegated-session-orchestrator
to run this as parallel sessions overnight.
```

### 3. Walk away

The orchestrator will:
1. Decompose the work into bounded task contracts
2. Dispatch sessions into separate worktrees
3. Run a heartbeat loop every 5 minutes
4. Review and merge completed sessions
5. Rescue stuck sessions by harvesting their commits
6. Dispatch the next batch when slots open up

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

`codex-orchestrator` is a practical **v1 loop**, not the final form of agentic
software development. It sits between manual prompting and a fully persistent
agent operating system.

| Level | Shape | What changes |
|-------|-------|--------------|
| **v0: Prompting** | Human prompts one agent at a time | The human is the scheduler, reviewer, and recovery loop |
| **v1: Supervised orchestrator skill** | `codex-orchestrator` today | Worktree isolation, bounded task contracts, heartbeat monitoring, review/merge discipline, evidence labels |
| **v2: Persistent task ledger** | A real state store behind the loop | Tasks, attempts, worker state, gates, blockers, and outcomes survive across threads and restarts |
| **v3: Routine library** | Reusable background routines | PR reviewer, CI fixer, stale-session rescuer, rebase helper, docs drift checker, release verifier |
| **v4: Eval and safety layer** | Failures become tests and policies | Prompt-injection cases, dangerous-operation classifiers, permission checks, evidence-quality evals |
| **v5: Agent operating system** | Many routines coordinate continuously | The human talks to loops/routines, while specialized agents execute, review, secure, and report |

This repository intentionally starts at v1 because that is the layer most teams
can adopt today without running a custom daemon or changing their whole
development platform. The next hard problems are durable state, routine
composition, safety classification, and eval-driven improvement.

The ambition is not to claim that a Codex skill is already a complete loop
runtime. The ambition is to make the first useful loop concrete: bounded work,
isolated execution, heartbeat inspection, honest proof labels, and review before
merge.

See [docs/v2-persistent-ledger-and-heartbeat.md](docs/v2-persistent-ledger-and-heartbeat.md)
for the first v2 seed: a durable ledger format and read-only heartbeat checker.
For the broader v2-v5 plan, see [docs/roadmap.md](docs/roadmap.md).

The v2 helper CLI currently supports:

```bash
go build -o codex-orchestrator ./cmd/codex-orchestrator
./codex-orchestrator init
./codex-orchestrator observe
./codex-orchestrator status
```

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
| **Evidence honesty** | Trust the session's self-report | `direct`/`proxy`/`blocked` labels enforced |
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
| Evidence labels | `direct`, `proxy`, `blocked` | Required classification for hardware/deploy/payment proof |
| Anti-shallow-slice | Enforced | Tasks must be classified before dispatch |

## 📂 File Structure

```
codex-orchestrator/
├── SKILL.md              # The orchestrator skill (copy to ~/.codex/skills/)
├── agents/
│   └── openai.yaml       # Agent interface definition
├── cmd/
│   └── codex-orchestrator/
│       └── main.go       # Go helper CLI seed
├── docs/
│   ├── roadmap.md
│   └── v2-persistent-ledger-and-heartbeat.md
├── examples/
│   └── ledger.example.json
├── scripts/
│   └── ledger_heartbeat.py
├── go.mod
├── README.md             # This file
├── README.zh-CN.md       # Chinese README
└── LICENSE               # MIT
```

## 📄 License

MIT

---

Built by [IndieKit.ai](https://indiekit.ai) — open-source developer tools for the AI-native workflow.
