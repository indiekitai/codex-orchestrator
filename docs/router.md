# Router Guide

`codex-orchestrator` works best when long-running work has a simple thread
topology instead of one overloaded chat. The Router is the smallest useful
piece of that topology: it decides where new input belongs.

The Router does **not** implement code, dispatch workers, merge branches, push,
or clean worktrees. It reads the thread map, classifies the input, and forwards
or summarizes it for the right owner.

## Thread Roles

| Thread | Owns | Does Not Own |
|---|---|---|
| Project Orchestrator | Repo truth, ledger, package plan, worker dispatch, review, merge, push, cleanup, status closeout | Raw issue collection, casual notes, unrelated research |
| Pulse | Scheduled status checks, missed-heartbeat reporting, material-change summaries | Implementation, merge decisions, new task invention |
| Inbox | GitHub issues, user feedback, external reviews, field notes, model/research links | Dispatching workers before triage |
| Router | Classifying new input and choosing the target thread | Code changes, worker control, merge/push/cleanup |
| Log | Human-readable decisions, package milestones, why a route changed | Authoritative ledger state |

## Routing Rules

1. Read `.codex-orchestrator/thread-map.md` first.
2. If the input is about current worker state, route to Project Orchestrator.
3. If the input is a scheduled check or missed heartbeat, route to Pulse.
4. If the input is new feedback, an issue, an external review, or research,
   route to Inbox.
5. If the input is a decision already made, append it to Log or tell Project
   Orchestrator to record it in the ledger/status surface.
6. If the input needs human action, device access, production/pre access,
   payment, SMS/provider, DNS/SSL, or deploy windows, route it as blocked or
   owner-gated. Do not dispatch a worker.
7. If the input is ambiguous, prefer Inbox with a short classification note
   rather than sending the Project Orchestrator into implementation.

## Minimal Router Prompt

```text
You are the Router for this codex-orchestrator project.

Read .codex-orchestrator/thread-map.md, .codex-orchestrator/inbox.md,
.codex-orchestrator/status.md, and the latest ledger/status truth if present.

Classify the new input into one of:
- Project Orchestrator
- Pulse
- Inbox
- Log
- Human/owner-gated

Do not edit code, dispatch workers, merge, push, cleanup, deploy, or create
new worker sessions. Output the target thread, reason, evidence label
(local/proxy/direct/blocked), and the exact message that should be forwarded.
```

## Router Output Shape

```text
Target: Inbox
Reason: New external feedback; not yet a task contract.
Evidence: local
Forward:
Please triage this feedback, link it to an existing feature package if
possible, and only propose a worker after checking the package lane and
current ledger state.
```

## Anti-Patterns

- Using the Router as a second Project Orchestrator.
- Letting the Router dispatch workers because capacity exists.
- Embedding current task IDs in Router prompts instead of reading ledger/status.
- Treating Inbox notes or social feedback as approved roadmap changes.
- Treating Pulse missed-heartbeat evidence as direct proof of the root cause.

Router discipline is intentionally boring. It keeps the product loop legible:
new input enters Inbox, active work stays with the Project Orchestrator,
scheduled checks stay in Pulse, and durable decisions become Log/ledger state.
