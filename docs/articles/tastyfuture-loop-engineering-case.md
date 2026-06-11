# From AI Coding Chats To A Supervised Engineering Loop

Date: 2026-06-11

Evidence boundary: `local/case-study`. This article is based on repository
docs, local review notes, and the sanitized TastyFuture orchestration case
study in this repository. It does not claim direct production, payment,
hardware, device, pre/prod, daemon, SEO, adoption, or external runtime proof.

## The Problem Was Not "More Agents"

Ad-hoc AI coding works well while the work is small. One Codex thread can read a
file, make a change, run a test, and hand back a diff. The failure mode appears
when the work becomes a queue:

- one feature has several independent slices;
- some slices touch central contracts and must not run in parallel;
- one worker finishes cleanly but waits for review;
- another worker stalls with useful uncommitted work;
- documentation, roadmap state, and evidence labels drift after merges;
- hardware, payment, provider, or pre/prod proof is genuinely unavailable.

Adding more agents does not solve that by itself. It can make the queue harder
to trust. The missing layer is a supervised loop around the agents: one that
decides what to dispatch, what to isolate, what to monitor, what to review, what
to merge, what to clean up, and what must remain blocked.

That is the practice behind `codex-orchestrator`.

## What Changed In The TastyFuture Run

TastyFuture was a useful case because it looked like real product work, not a
toy demo. The project involved restaurant POS, web/cloud, mobile, payment, and
operational workflows. Several tasks could be moved forward locally, but some
proof surfaces were outside the agent's reach: payment terminals, physical
devices, human-action steps, provider callbacks, and pre/prod environments.

The orchestrator pattern made the outer loop explicit.

### 1. A Worker Got A Contract, Not A Vague Mission

A delegated task was framed as a bounded contract:

- task id and expected outcome;
- allowed paths and forbidden paths;
- base branch or base commit;
- verification gates;
- required review notes;
- evidence labels;
- blocked stop condition.

That made each worker branch reviewable. A worker could still make mistakes,
but the reviewer had a concrete contract to compare against the diff.

### 2. Worktree Isolation Became The Unit Of Trust

Each worker ran in an isolated Codex App worktree session. That mattered more
than the chat summary. The useful questions became:

- Which worktree exists?
- Which branch is checked out?
- Is it dirty?
- Are there commits after the base commit?
- Did it touch forbidden paths?
- Did it run the requested gates?
- Did it write a self-review and evidence boundary?

This turned multi-session work from a pile of conversations into local git
state that could be inspected.

### 3. Heartbeat Was Reconciliation, Not Status Theater

The heartbeat was useful only when it rediscovered truth from the repository,
ledger, and worktrees. A good heartbeat did not ask "what did the last message
say?" first. It checked:

- current branch and worktree state;
- pending setup versus real resolved worktree;
- active dirty work;
- clean completed commits waiting for review;
- blocked tasks;
- cleanup-needed branches;
- queue capacity and whether the loop should continue.

This is why the project emphasizes a durable ledger and status reports. Chat is
too fragile to be the only state store for long-running orchestration.

### 4. `completed-unreviewed` Became A First-Class State

One important TastyFuture lesson was that a clean worker commit is not the same
as accepted work. It is also not nothing.

Treating that state as `completed-unreviewed` kept the queue moving without
overclaiming. The orchestrator could review the diff, run local checks, inspect
evidence labels, decide whether central docs needed updates, and only then
merge.

### 5. Review, Merge, Push, And Cleanup Stayed Reviewer-Owned

`codex-orchestrator` is intentionally not an unreviewed autonomous coding bot.
The review loop remained explicit:

1. reread the task contract;
2. inspect changed paths against allowed and forbidden boundaries;
3. read the worker's self-review;
4. run or inspect verification gates;
5. classify evidence as `direct`, `proxy`, `local`, or `blocked`;
6. update docs when central project state changed;
7. merge only if the evidence supported it;
8. push only when that matched normal repo policy;
9. clean up only after the branch/worktree no longer carried useful state.

The helper can produce local/static review packets, but the acceptance decision
is still an engineering decision.

## Practical Workflow

This is the working loop a new user should try.

### Dry Run

Paste the Quick Start prompt into Codex App from the repository you want to
orchestrate. Ask for read-only analysis first.

Expected output: a proposed task split, worktree plan, evidence boundaries,
review gates, and destructive-action limits. No worker creation, merge, push,
or cleanup should happen during the dry run.

Evidence label: `local/static`.

### Dispatch

After review, approve a small number of bounded tasks. Use separate worktrees
and branches. Record the task contract, allowed paths, forbidden paths, gates,
and pending setup state if the helper is available.

Evidence label: `local/setup` until a real worktree and branch exist. A
`pendingWorktreeId` is not proof that a worker is running.

### Monitor

Use heartbeat/status checks to reconcile the live queue with repo truth. The
monitor should separate active work, dirty progress, pending setup,
completed-unreviewed commits, blocked tasks, and cleanup-needed worktrees.

Evidence label: `local/static`.

### Review

When a worker has a clean commit, review it as a delivery unit. Check the diff,
paths, self-review, gates, docs drift, and evidence labels.

Evidence label: usually `local`. Only call something `direct` when the task
artifact includes actual direct proof for that surface.

### Merge

Merge accepted work through normal repo policy. If a task has only local proof,
say so. If payment, hardware, provider, device, pre/prod, or human-action proof
was not exercised, keep it blocked.

Evidence label: `local` for the merge decision unless stronger proof is truly
present.

### Cleanup

Clean worktrees only after review closure. Do not delete dirty or ambiguous
work because a chat summary says the task is done.

Evidence label: `local`.

### Continue Or Stop

After each accepted task, check the broader queue before stopping. The loop is
not complete just because one worker merged. It is complete when the queue is
drained, intentionally paused, blocked by named evidence, or waiting for human
approval.

Evidence label: `local/blocked` depending on the reason.

## Evidence Labels Are The Product Discipline

The TastyFuture case made evidence labels non-negotiable:

- `direct`: the relevant surface itself was exercised and recorded;
- `proxy`: indirect evidence supports the claim but does not prove the final
  surface;
- `local`: repo, source, test, static, ledger, routine, or local workflow
  evidence;
- `blocked`: the necessary environment, credential, device, provider, payment,
  deployment, or human action was not available.

The rule is conservative: never upgrade local or proxy evidence into direct
runtime, production, device, payment, or hardware proof.

## What This Is Not

`codex-orchestrator` is not:

- a daemon that runs Codex sessions by itself;
- a package-manager-first product where the user must learn a helper CLI before
  trying it;
- a full agent operating system;
- a replacement for code review;
- a guarantee that worker output is correct;
- a production, hardware, payment, or device verification system;
- a way to hide blocked proof behind confident wording.

It is a Codex App-first harness: task contracts, isolated worker sessions,
ledger/status visibility, routines, review packets, evidence labels, and
continuation rules around human-supervised engineering work.

## Limits And Open Edges

The repository already contains local/static evidence for the workflow shape:
case notes, review notes, helper commands, policy/eval fixtures, and routine
docs. That is useful, but it has limits.

Blocked or not claimed here:

- external adoption impact;
- SEO or discovery impact after publication;
- production runtime behavior;
- real payment capture;
- physical device acceptance;
- pre/prod environment validation;
- standalone scheduler or daemon behavior;
- direct Codex App API control beyond App-mediated workflow practice.

Those are not wording gaps. They are boundaries. A loop that keeps them visible
is more useful than a bot that pretends they disappeared.

## Source Trail

Start with these repo-local materials:

- [TastyFuture Orchestration Case Study](../case-studies/tastyfuture-orchestration.md)
- [TastyFuture Orchestration Feedback](../reviews/2026-06-11-tastyfuture-orchestration-feedback.md)
- [Loop Engineering Alignment](../research/loop-engineering-alignment.md)
- [Agentic Engineering Feature Notes](../research/agentic-engineering-feature-notes.md)

Together they show the local case evidence, the product implications, and the
reason this project stays focused on supervised Loop Engineering instead of an
agent OS or unreviewed autonomous coding.
