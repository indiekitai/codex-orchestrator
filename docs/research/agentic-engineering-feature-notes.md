# Agentic Engineering Feature Notes

This note records the product implications from recent Agentic Engineering and
Agentic Software Engineering materials. It is intentionally separate from the
implementation roadmap: these are feature candidates and positioning
constraints, not claims about completed functionality.

## Source Alignment

Recent materials point to the same direction:

- LangChain's April 2026 Agentic Engineering article frames the field as a
  multi-agent coordination model with defined roles, shared memory, and common
  observability across the software delivery pipeline.
- The SASE paper, *Agentic Software Engineering: Foundational Pillars and a
  Research Roadmap*, introduces an Agent Command Environment (ACE), where
  humans orchestrate and mentor agent teams, and an Agent Execution Environment
  (AEE), where agents perform bounded work and call back for human judgment.
- The later arXiv framing around agentic software emphasizes a role shift from
  code author to intent architect, coordinator, and outcome auditor.

`codex-orchestrator` maps most directly to a lightweight, Codex App-first ACE:
it helps a human-led orchestrator split work, isolate execution, maintain
state, inspect progress, review evidence, and decide whether to merge or ask
for help.

The current project is not a full AEE, daemon, worker pool, or agent operating
system. Codex App still owns worker session creation and execution. The helper
CLI provides local state, reports, routines, and policy/eval checks around
that App-first workflow.

## Functional Implications

The next useful features should make orchestration outputs more standard and
auditable. The goal is not to add more agents. The goal is to make each agent's
handoff easier to trust, reject, or escalate.

### 1. Ledger-Enforced Dispatch Closure

Purpose: make task dispatch and setup reconciliation durable immediately,
instead of leaving pending worktree ids and worker setup facts in chat text.

Possible command shape:

```bash
codex-orchestrator dispatch record --task-id TASK --pending-worktree-id ID
codex-orchestrator dispatch reconcile --task-id TASK
```

Expected task state:

- task id and title;
- pending worktree id returned by Codex App, when available;
- resolved thread id, worktree path, branch, and base commit;
- allowed paths and forbidden paths;
- expected gates;
- current lifecycle state: pending setup, active, dirty-uncommitted,
  completed-unreviewed, blocked, merged, or cleaned;
- latest setup event and latest reviewer event.

Evidence boundary:

- This is `local/static` orchestration state.
- A pending worktree id is not direct proof that a worker is running.
- A reconciled worktree path is not proof that the task is correct.

Why it matters:

- Real TastyFuture runs still relied on heartbeat text and chat summaries for
  pending ids and setup facts.
- Heartbeats should recover from ledger truth before reading stale prompt text.
- This is the base layer for reliable long-running App-first orchestration.

### 2. Merge-Readiness Pack

Purpose: turn "the worker says it is done" into a standardized review packet.

Possible command:

```bash
codex-orchestrator pack merge-readiness --task-id TASK
```

Expected output:

- task id, title, branch, worktree, base commit, latest commit;
- `git diff --name-status baseCommit..HEAD`;
- allowed-path and forbidden-path check result;
- `git diff --check` result;
- review doc and artifact presence;
- worker self-review presence;
- gates requested vs gates observed;
- evidence labels found in docs/review/handoff;
- docs drift signal;
- residual risks and blocked claims;
- recommended orchestrator action: review, reject, merge, ask human, or block.

Evidence boundary:

- This is `local/static` review evidence.
- It must not claim runtime, production, hardware, payment, or device proof.
- It can include references to direct proof only when that proof is already
  present in worker artifacts and clearly labeled.

Why it matters:

- It matches the SASE "Merge-Readiness Pack" shape.
- It gives future UI/status layers a concrete object to render.
- It reduces dependence on long chat handoffs.
- It captures the orchestrator's acceptance reasoning, not only the worker's
  self-review.

### 3. Consultation Request Pack

Purpose: turn a blocker or human decision into a concise, actionable request.

Possible command:

```bash
codex-orchestrator pack consultation --task-id TASK
```

Expected output:

- what is blocked;
- why the agent cannot safely continue;
- what was tried;
- local/proxy/direct evidence already gathered;
- what human input or physical action is needed;
- safe choices the user can make;
- consequences of each choice;
- whether worktree/branch should be kept, retried, or cleaned later.

Evidence boundary:

- A consultation pack should default to `blocked` plus any supporting
  `local`, `proxy`, or `direct` evidence.
- It must not hide uncertainty behind a confident recommendation.

Why it matters:

- It matches the SASE "Consultation Request Pack" shape.
- It fits real hardware/payment workflows where the agent needs a human to
  unplug a printer, confirm a PAX prompt, approve a deployment window, or make
  a product decision.
- It gives the human a better interface than a scattered chat explanation.

### 4. Project-Aware Roadmap Scorer

Purpose: prevent the loop from drifting into low-value readiness pages just
because they are safe and easy to dispatch.

Possible command:

```bash
codex-orchestrator roadmap score --repo .
```

Expected inputs:

- configurable project source-of-truth docs, such as `PROGRESS.md`,
  roadmap files, package plans, or project-specific planning docs;
- selected review docs only when explicitly configured, because review notes
  often contain risks and postmortems rather than dispatchable backlog;
- existing ledger state;
- recent cleaned/merged/blocked tasks;
- forbidden external/hardware/provider/pre/prod boundaries.

Expected output:

- candidate task id/title;
- source document and evidence snippet;
- classification: `vertical-completion`, `runtime-proof`,
  `blocked-removal`, `owner-gated`, or `shallow-risk`;
- write-set risk;
- external dependency risk;
- suggested action: dispatch, ask owner, block, or skip.

Evidence boundary:

- This is local/static planning evidence.
- It should not claim product completion or runtime proof.
- A high score is only a dispatch recommendation, not approval to merge.

Why it matters:

- In TastyFuture, many local readiness pages were valid but could hide the
  higher-value blocked work that actually moves the product forward.
- The tool should help the orchestrator avoid "safe but shallow" task churn.
- Feature-package candidates should rank ahead of unrelated safe task fillers,
  so the orchestrator keeps pushing one module to closure.
- This makes roadmap-next-task selection project-aware instead of hard-coded to
  `docs/roadmap.md`.

### 5. Transcript / Heartbeat Failure Eval

Purpose: convert orchestration mistakes into deterministic regression
fixtures.

Possible command shape:

```bash
codex-orchestrator eval run --suite orchestration-transcripts
```

Candidate fixtures:

- child task completes, but the orchestrator deletes the heartbeat and stops
  instead of checking the remaining queue;
- heartbeat text is bound to stale task ids instead of dynamically reading
  repo truth and ledger truth;
- `pendingWorktreeId` is treated as a running worker before a real worktree and
  branch exist;
- setup failure falls back to implementation in the main orchestrator checkout;
- worker local/proxy evidence is promoted to direct/pre/prod/device proof;
- orchestrator implements worker code itself after dispatch failure;
- worker final handoff lacks self-review, docs drift check, or evidence labels.

Why it matters:

- This makes the project self-improving without pretending rules can update
  themselves automatically.
- It connects real TastyFuture and codex-orchestrator failures to V4 policy/eval
  work.
- It creates a practical evaluation layer for "loop quality", not just code
  correctness.

### 6. Static Dashboard / Status Page

Purpose: make the current queue understandable without reading JSON or a long
thread.

Possible command:

```bash
codex-orchestrator status --html > orchestrator-status.html
```

Expected sections:

- repo status and integration checkout cleanliness;
- active tasks;
- pending setup;
- dirty-uncommitted worker progress;
- completed-unreviewed tasks;
- blocked tasks;
- cleanup-needed tasks;
- recently merged/cleaned tasks;
- available dispatch slots;
- budget/review pressure;
- next suggested action;
- evidence label summary.

Evidence boundary:

- This is `local/static` status evidence.
- It does not prove Codex App runtime delivery or real worker health beyond
  git/worktree/ledger facts.

Why it matters:

- It directly addresses the user problem: "what is happening now, and why did
  the loop not continue?"
- It gives future UI work a low-risk stepping stone without introducing a
  daemon or web server.

## Priority

Recommended order:

1. Ledger-Enforced Dispatch Closure.
2. Merge-Readiness Pack.
3. Project-Aware Roadmap Scorer.
4. Consultation Request Pack.
5. Transcript / Heartbeat Failure Eval.
6. Static Dashboard / Status Page.

The first item makes long-running state recoverable. The next two improve
review quality and task choice. Consultation packs improve human handoff.
Transcript evals make the orchestrator safer over time. The status page
improves usability and visibility.

## Non-Goals For This Phase

Do not use this research as justification for:

- a full daemon;
- automatic session scheduling outside Codex App;
- automatic merge/push/cleanup without human or orchestrator review;
- a worker pool or agent OS;
- package-manager distribution work;
- claims of production, hardware, payment, or device proof from local/static
  helper output.

## Public Positioning

Useful public wording:

> `codex-orchestrator` is a lightweight Agent Command Environment for Codex
> App. It does not replace coding agents; it gives a human orchestrator a
> repeatable way to assign bounded work, inspect isolated sessions, request
> merge-readiness or consultation packs, and keep evidence labels honest.

Avoid saying:

- "fully autonomous software engineering";
- "agent operating system";
- "production-grade agentic engineering platform";
- "Codex replacement";
- "direct proof" unless the proof is actually direct.
