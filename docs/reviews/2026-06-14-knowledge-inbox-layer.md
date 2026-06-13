# Knowledge / Inbox Layer

## Context

Follow-up from the thread-topology package. Public Codex workflow examples show
that durable threads are more useful when they can read a local task board and a
local concept library before routing work. This change keeps that idea local and
file-based instead of adding external Notion or remote sync.

## Changes

- `init --write-templates` now writes:
  - `.codex-orchestrator/concepts.md`
  - `.codex-orchestrator/inbox.md`
- `observe`, `status`, summaries, and preflight now expose `concepts` and
  `inbox` local/static readiness blocks.
- Router prompt templates now instruct the router to read the thread map,
  project map, concepts library, inbox, and current status before classifying
  input.
- README, Chinese README, SKILL, v2 docs, full guides, roadmap, and research
  notes now describe the local knowledge/intake layer.

## Evidence Labels

- `local/static`: template generation, status/preflight file detection, docs,
  tests, and policy/evidence checks.
- `proxy`: public workflow examples informed the design.
- `blocked`: no direct Codex App cross-thread messaging proof, Notion sync,
  external inbox import, or runtime automation-delivery proof is claimed.

## Boundaries

`concepts.md` and `inbox.md` are coordination files. They are not a task ledger,
not external knowledge-base sync, not direct proof, and not permission for a
Router/Inbox/Pulse thread to implement code, dispatch workers, merge, push,
deploy, or clean worktrees.
