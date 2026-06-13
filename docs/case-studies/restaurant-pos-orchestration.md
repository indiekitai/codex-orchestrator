# Restaurant POS Rewrite Orchestration Case Study

Date: 2026-06-11
Project type: large restaurant POS rewrite and adjacent web/cloud work
Evidence label for this page: `local/project-case-study`

This is a sanitized public case study based on real Codex App orchestration
feedback from a restaurant POS rewrite. It shows what `codex-orchestrator`
looked like when used against a large ongoing codebase. It does not claim
direct runtime, production, payment, hardware, daemon, or standalone session
runtime proof.

## Why this project was a useful test

The project had the failure modes that make single-thread AI work brittle:

- multiple feature slices moving at once,
- central contracts that could not be edited in parallel safely,
- review/docs drift after merges,
- local proof that was useful but not safe to overstate as production truth,
- human-action and environment boundaries that had to stay blocked.

That made it a good fit for a supervised outer loop instead of one long agent
thread.

## What the orchestrator concept changed

### 1. Worktree isolation instead of chat-window sprawl

Each worker ran in its own worktree and branch. That turned "go build this
slice" into a reviewable delivery unit with explicit file boundaries and gate
commands, instead of relying on vague thread summaries.

### 2. Repo truth before thread truth

The useful status signal came from repo state first:

- `git status --short --branch`
- `git worktree list --porcelain`
- recent commits
- ledger and routine reports when present

This mattered because child-thread text and actual git state did not always
agree.

### 3. Heartbeat as reconciliation, not narration

The heartbeat was valuable when it rediscovered current repo/worktree truth
instead of replaying stale task IDs. A useful loop was:

1. inspect active worktrees and branches,
2. classify pending setup, active work, completed-unreviewed commits, blocked
   work, and cleanup-needed state,
3. decide whether to wait, review, merge, or stop.

### 4. `completed-unreviewed` was a real state

One important behavior change was treating a clean worker commit as
`completed-unreviewed` even when the child thread looked ambiguous. That kept
the review queue moving without pretending the task was already accepted.

### 5. Merge/push/cleanup stayed reviewer-owned

The helper could support ledger truth and read-only checks, but the actual
decisions still belonged to the reviewing Codex App orchestrator:

- reread the diff,
- check allowed and forbidden paths,
- inspect self-review quality,
- run gates,
- merge only if the result was credible,
- push only when normal for the repo,
- clean the worktree only after review closure.

## Evidence labels mattered

The restaurant POS rewrite was the kind of project where evidence drift would be dangerous.
Many slices had useful closure, but not the same kind of closure.

- `local`: source, test, git, routine, and local workflow proof
- `proxy`: indirect system or release evidence when explicitly labeled
- `blocked`: payment, device, hardware, network, pre/prod, or human-action
  boundaries that were not actually exercised

The important rule was negative: do not upgrade local or proxy proof into
direct runtime, pre, prod, payment, or hardware claims.

## What stayed blocked on purpose

This workflow still relied on explicit blocked boundaries for:

- payment terminals and payment capture,
- printers and other hardware,
- real device acceptance,
- production or pre-production environment claims,
- human-action steps that required someone at the machine or in the store.

That is a feature, not a gap in wording. The case study is useful because those
limits stayed visible.

## Practical takeaways for a new user

If you want to try `codex-orchestrator`, the lesson is simple:

1. Give Codex App the GitHub repository.
2. Let Codex read `README.md`, `SKILL.md`, and the setup docs.
3. Ask for a dry run first.
4. Let Codex decide whether the helper is useful for ledger/routine support.
5. Keep merge/push/cleanup and evidence review explicit.

This is the intended product path. It is not "install a CLI first and then
learn a daemon."

## Boundaries of this case study

Confirmed here:

- local workflow evidence for worktree isolation,
- repo-truth-first review flow,
- heartbeat/review/merge/cleanup concepts,
- evidence-label discipline,
- blocked-boundary discipline.

Not claimed here:

- direct deployed runtime proof,
- production proof,
- payment proof,
- hardware proof,
- standalone daemon proof,
- fully autonomous session scheduler proof.

For the raw local feedback that informed this case study, see
`docs/reviews/2026-06-11-restaurant-pos-orchestration-feedback.md`.
