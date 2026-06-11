# Harness Reading Roadmap Review

Date: 2026-06-11

Scope:

- `README.md`
- `README.zh-CN.md`
- `docs/roadmap.md`
- `docs/research/loop-engineering-alignment.md`
- `docs/research/harness-reading-notes.md`

Inputs:

- `/Users/tf/Downloads/book1-claude-code.pdf`
- `/Users/tf/Downloads/book2-comparing.pdf`

Summary:

The two local books reinforced that `codex-orchestrator` should be described as
a Codex App-first harness for Loop Engineering, not as a future agent operating
system. The useful product boundary is the control plane around Codex App
sessions: task contracts, isolated worktrees, durable state, heartbeat truth
checks, recovery paths, verification routines, policy/eval checks, and
reviewable rule improvement.

Changes recorded:

- Added `docs/research/harness-reading-notes.md` with the reading synthesis.
- Updated README positioning from "supervised outer loop" to "Codex App-first
  harness for Loop Engineering."
- Updated the Chinese README with the same product boundary.
- Removed the old agent-OS roadmap stage.
- Added an explicit "Agent operating system is out of scope" boundary.
- Reframed next development around V4 policy/eval, recovery-state
  classification, selective routines, and App-first adoption.

Evidence labels:

- `local`: PDF text was reviewed from local files.
- `local`: documentation was updated in this repository.
- `blocked`: no external source refresh was performed in this change; this was
  a local reading and roadmap update, not a new web research pass.

Verification:

- Planned: `rg` check for stale agent-OS roadmap-stage references.
- Planned: docs drift / evidence label / policy routines.
- Planned: `git diff --check`.

Residual risk:

- The research notes summarize copyrighted local PDFs without quoting long
  passages. If these notes become public marketing copy, keep them as product
  conclusions rather than source excerpts.
