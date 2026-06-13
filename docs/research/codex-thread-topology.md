# Codex Thread Topology Notes

Date: 2026-06-14

## Why This Matters

Long-running Codex App work is starting to look less like one chat and more like
a small operating system made of durable threads. Dan Shipper described a
practical pattern with Pulse, Log, Inbox, and Router threads:

- Pulse threads wake on a schedule and check specific areas.
- Log threads keep ongoing activity history.
- Inbox threads collect important inputs.
- Router threads know about the other threads and route input to the right
  place.

The important lesson is not "open more chats." The lesson is that durable
threads need both memory and control flow. Pulse/Log/Inbox provide memory.
Router provides control flow. Without explicit routing, the user still has to
manually remember which thread owns which context.

This aligns with Codex App user feedback asking for task boards,
cross-thread references, dispatcher/controller threads, and handoff between
durable threads.

Long Chen's follow-up framing adds one product detail worth keeping: the
handoff layer is not only thread roles. It is also a local task board plus a
concept library. A Router is much more useful when it can read stable local
state before deciding where to send input:

- task board / ledger / status / reports: what is happening now;
- project map: where code and ownership live;
- thread map: which durable thread owns which role;
- concepts: stable terms, rules, decisions, and historical pitfalls;
- inbox: untriaged issues, user feedback, external reviews, pulse outputs, and
  run observations.

## Product Interpretation

`codex-orchestrator` should remain a project-level engineering orchestrator,
not a general agent operating system. But it should make long-lived Codex App
thread layouts and local knowledge handoff explicit and reviewable.

The minimum useful model is:

| Role | Responsibility | Must not do |
|---|---|---|
| Project Orchestrator | repo truth, ledger, worker dispatch, review, merge, push, cleanup, package closeout | route every unrelated personal/work inbox item |
| Pulse | scheduled read-only status checks and missed heartbeat reporting | implement code, merge, push, cleanup |
| Inbox | collect issues, external review, user feedback, run observations | dispatch workers directly |
| Router | classify new input and generate handoff prompts for the right owner thread | implement code, dispatch workers, merge, push, deploy, cleanup |
| Log | human-readable operating journal | act as source of truth over repo/ledger |

The local knowledge layer is intentionally small:

| File | Purpose | Evidence boundary |
|---|---|---|
| `.codex-orchestrator/concepts.md` | glossary, stable rules, prior decisions, pitfalls, blocked concepts, source docs | local/static |
| `.codex-orchestrator/inbox.md` | intake for feedback, issues, external reviews, pulse outputs, run observations | local/static |

## Design Decisions

1. Add thread topology as local/static project state, not global memory.
   - File: `.codex-orchestrator/thread-map.md`
   - Status field: `threadMap`
   - Preflight warning when missing.

2. Add reusable prompt templates for recurring thread roles.
   - File: `.codex-orchestrator/pulse-threads.md`
   - Covers Project Pulse, Inbox, Router, and Log.

3. Keep Router out of execution.
   - Router can classify and hand off.
   - Router cannot implement, dispatch, merge, push, deploy, or cleanup unless
     explicitly promoted to Project Orchestrator.
   - Guarded by policy/eval rule `OPA011`.

4. Keep the core product focused.
   - No attempt to build a full cross-thread task board yet.
   - No automated cross-thread messaging protocol yet.
   - No promise that thread-map entries prove live thread or automation state.

5. Add Concepts and Inbox as files, not integrations.
   - File: `.codex-orchestrator/concepts.md`
   - File: `.codex-orchestrator/inbox.md`
   - Status fields: `concepts`, `inbox`
   - Preflight warnings when missing.
   - No Notion/remote sync yet; local Markdown stays inspectable and portable.

## Evidence Boundary

- `local/static`: thread-map, concepts, or inbox file exists; status/preflight
  can read it.
- `proxy`: public posts and discussions describing durable-thread patterns.
- `blocked`: no direct Codex App runtime proof that a Router can message every
  thread or that automations fired on schedule.

Thread topology is coordination state. The orchestrator must still verify live
thread ids, recent messages, automation bindings, repo truth, worktree truth,
and ledger state before mutating anything.

## Next Possible Work

- Add `thread-map validate` if thread-map structure becomes more formal.
- Add `concepts validate` or lightweight lint if the concepts file becomes
  structured enough to check stale decisions.
- Add read-only inbox import after GitHub/X/Codex App thread-list/message APIs
  become stable.
- Add status UI grouping for project orchestrator / pulse / inbox / router
  if thread metadata becomes machine-readable.
