# v0.3.1 Release Notes

`v0.3.1` is a docs and visibility release. It does not change the helper's
runtime behavior. The goal is to make the project easier to understand from the
GitHub first screen and to show a concrete, evidence-labeled real-project case.

## Highlights

- Reworked the README hero around the actual product shape:
  Codex App-first Loop Engineering for real repositories.
- Made the boundary explicit: this is not a daemon, package-manager-first
  install path, full agent OS, or unreviewed autonomous coding bot.
- Added a clearer first-use path: paste the Quick Start prompt into Codex App,
  let Codex read the repository, install/update the skill if needed, and start
  with a dry run.
- Updated the Chinese README with a more natural explanation of the same
  positioning.
- Added a TastyFuture case article:
  `docs/articles/tastyfuture-loop-engineering-case.md`.
- Updated roadmap status to mark the App-first README explanation and
  TastyFuture case/bootstrap docs as completed.

## Case Article

The new article explains why the problem is not simply "more agents." The
useful layer is a supervised engineering loop around Codex worker sessions:

- bounded task contracts;
- isolated Codex App worktree sessions;
- repo-truth heartbeat reconciliation;
- `completed-unreviewed` as a first-class state;
- reviewer-owned review / merge / push / cleanup;
- conservative `direct`, `proxy`, `local`, and `blocked` evidence labels.

Evidence boundary: the article is `local/case-study` evidence based on
repository docs and local review notes. It does not claim external adoption,
SEO, production, payment, hardware, device, pre/prod, daemon, or runtime proof.

## Install / Trial Path

The recommended trial path remains Codex App-first:

```text
I want to try codex-orchestrator in this repository.

Read https://github.com/indiekitai/codex-orchestrator and use it as a
Codex App-first orchestration workflow.

If the Codex App skill from that repository is not installed, install it into
~/.codex/skills/codex-orchestrator.

Start with a dry run:
- inspect git status, worktrees, and project docs;
- explain how you would split work into isolated Codex worktree sessions;
- explain what you would monitor, review, merge, push, and clean up;
- label evidence as direct, proxy, local, or blocked.

Do not push, deploy, delete worktrees, or make destructive changes unless I
explicitly approve.
```

The Go helper remains optional support for local ledger state, status reports,
heartbeat summaries, and routine checks. Users should not need to learn the CLI
before trying the workflow in Codex App.

## Verification Before Publishing

Docs/review checks used for this release:

- `git diff --check`
- local Markdown link/path sanity checks
- `codex-orchestrator pack merge-readiness`
- `codex-orchestrator run-routine pr-reviewer`
- `codex-orchestrator run-routine docs-drift-checker`
- `codex-orchestrator run-routine evidence-label-auditor`

No source/runtime changes were made in this release.

## Boundaries

This release does not:

- create Codex App sessions from the CLI,
- run as a background daemon,
- merge or push automatically from the helper,
- clean worktrees automatically,
- replace human engineering review,
- add Homebrew/npm/tap/package-manager distribution,
- prove production, hardware, payment, device, pre/prod, SEO, adoption, or
  deployed runtime behavior.

## Suggested Announcement

`codex-orchestrator v0.3.1` tightens the GitHub first screen and adds a
TastyFuture case article. The project is now easier to explain: Codex App-first
Loop Engineering with task contracts, isolated worktree sessions, heartbeat
status, review-before-merge, cleanup discipline, and honest evidence labels.
