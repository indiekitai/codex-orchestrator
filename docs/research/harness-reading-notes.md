# Harness Reading Notes

This note records what the project learned from two local PDF books reviewed on
2026-06-11:

- `/Users/tf/Downloads/book1-claude-code.pdf`
- `/Users/tf/Downloads/book2-comparing.pdf`

The notes below are a product and roadmap synthesis, not a transcript or a
verbatim summary of the books.

## Core Lesson

The useful abstraction is not "a smarter prompt" or "more agents." The useful
abstraction is a **harness** around coding agents:

- feedforward guidance before the agent acts;
- tool and permission boundaries while it acts;
- feedback sensors that prove what happened;
- recovery paths when the loop stalls or fails;
- durable state outside the transient chat;
- independent review before trusting or shipping results.

That maps directly to Loop Engineering: the engineer is not only prompting an
agent, but designing the loop that prompts, observes, verifies, recovers, and
learns.

## Later Note: Harness As Executable Judgment

A useful refinement from later Harness / AI-first organization discussions is
that a harness is not just a toolkit around an agent. MCP servers, skills,
schemas, sandboxes, logs, traces, and local CLIs are all useful, but they are
not enough by themselves.

The missing layer is the project's own definition of "what counts as right":

- which sources are authoritative;
- which boundaries the agent may cross;
- which output shape is machine-checkable;
- which evidence is strong enough for a claim;
- which ambiguous cases require a human or owner decision;
- how review feedback becomes a rule, fixture, spec, or status improvement.

For `codex-orchestrator`, this means worker contracts must carry an acceptance
definition, source-of-truth context, allowed/forbidden paths, gates, evidence
labels, and blocked conditions. Acceptance reports must explain why the task
meets that definition; they should not rely on task count, clean git status, or
the worker's self-report alone.

If these judgments are not written into the harness, an agent tends to borrow
generic defaults. The result can be polished, structured, and still wrong for
the project. The product should keep optimizing for making project-specific
judgment visible, executable, checkable, and revisable.

## Book 1: Claude Code / Harness Engineering

The first book reinforces a practical point: the agent's reliability is mostly
shaped by the harness around it.

Useful takeaways for `codex-orchestrator`:

- A prompt is part of the control plane, not just personality text.
- Tool calls are managed execution interfaces. Dangerous tools need explicit
  rules, approval points, and failure handling.
- Errors and recovery are main-path behavior. A long-running loop should assume
  stale workers, dirty worktrees, missing commits, failed setup, and ambiguous
  evidence will happen.
- Verification quality matters more than the number of skills or routines.
  A loop that cannot observe runtime behavior is still mostly a task manager.
- Multi-agent work needs role separation, state isolation, lifecycle hooks, and
  independent review.
- Hooks and automation are advanced surfaces. They should come after the basic
  control plane is stable, not before it.

Implication: `codex-orchestrator` should keep investing in task contracts,
isolated worktrees, heartbeat truth checks, review gates, and recovery states
before trying to become a larger agent platform.

## Book 2: Claude Code vs Codex Harness Design

The second book is useful because it distinguishes two harness philosophies.

The relevant product reading:

- Claude Code is strong as a runtime-first, query-loop-first field tool. It is
  good at entering a local codebase, running commands, inspecting failures, and
  iterating in place.
- Codex is strong when the control plane is explicit: local rules, skills,
  structured tools, thread/worktree state, policy boundaries, and auditable
  handoffs.
- For this project, Codex App is the important surface because it can create
  and continue isolated sessions and coordinate worktrees. The helper CLI should
  support that control plane, not pretend to replace it.

Implication: the public positioning should be **Codex App-first harness for
Loop Engineering**, not "multi-agent OS" and not "a standalone daemon that
does all engineering work."

## What This Changes

The earlier roadmap was directionally right, but over-weighted a possible
"agent operating system." The reading makes that the wrong route for this
project.

Near-term work should focus on:

1. **Policy and eval**
   - Preserve real failure cases as fixtures.
   - Detect evidence over-claiming, unsafe fallbacks, destructive actions, and
     missing approval boundaries.
   - Keep rule updates review-only until they are proven.

2. **Recovery state model**
   - Model setup-failed, stale-pending, dirty-worker, completed-unreviewed,
     blocked, cleanup-needed, interrupted, and abandoned states explicitly.
   - Make stale task rescue predictable instead of conversational.

3. **Verification routine contracts**
   - Keep routines as bounded workflow contracts with named inputs, sensors,
     gates, evidence output, and escalation rules.
   - Avoid adding routine names unless they remove a real repeated failure.

4. **App-first onboarding**
   - A new user should be able to paste one prompt into Codex App.
   - Codex App should read the repository, install the skill if needed, explain
     the helper, and dry-run before mutating anything.

5. **Living runbook**
   - Repeated orchestration failures should become docs, fixtures, rules, or
     skill changes.
   - The loop should improve by evidence, not by vague "be careful" reminders.

## What Not To Build

Do not build a multi-agent operating system in this project.

Reasons:

- There is no stable public worker-control surface that lets the helper own
  Codex App sessions directly.
- A daemon or UI would not solve the current hardest problems: evidence quality,
  review discipline, recovery classification, and policy/eval.
- A larger platform would make the project harder to adopt before the App-first
  harness path is polished.
- The value proposition is already strong if the project reliably turns Codex
  App into a supervised, auditable, recoverable engineering loop.

If a broader worker runtime is ever worth building, it should be a separate
product decision with its own user model and safety model, not the next version
of this repository.

## Updated Product Positioning

Use this framing:

> `codex-orchestrator` is a Codex App-first harness for Loop Engineering. It
> helps a supervising Codex session split roadmap work into isolated worktree
> sessions, track state in a local ledger, run heartbeat and policy checks,
> review evidence, merge clean branches, and recover stalled work.

Avoid this framing:

- autonomous engineering OS;
- fully automatic development team;
- daemon-first agent platform;
- proof that Codex App can be left unattended without review;
- a replacement for runtime verification.

## Next Development Route

The next route is:

1. Finish V4 as the product's main near-term track:
   - expand policy/eval fixtures from real orchestration failures;
   - add recovery-state classifier coverage;
   - add review-only rule proposal support;
   - strengthen evidence-label checks.

2. Keep V3 routines selective:
   - add or refine a routine only when it covers a real recurring workflow;
   - require every routine to name its verification surface and escalation
     boundary.

3. Improve App-first adoption:
   - keep README focused on the one-prompt Codex App path;
   - keep manual CLI installation secondary;
   - maintain clean-clone and real Codex App demo proof.

4. Keep agent OS out of scope:
   - no agent OS claim;
   - no daemon that mutates git/session state;
   - no broad worker pool in this repository.
