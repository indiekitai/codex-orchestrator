# README Hero And Case Article Review

Date: 2026-06-11
Task ID: TF-CODEX-ORCH-DOCS-README-HERO-CASE-ARTICLE-LOCAL

## Changed Files

- `README.md`
- `README.zh-CN.md`
- `docs/articles/tastyfuture-loop-engineering-case.md`
- `docs/reviews/2026-06-11-readme-hero-case-article.md`

## Evidence Basis

- `local`: current README files, repository case-study docs, local review docs,
  and research notes.
- `proxy`: none.
- `direct`: none. This docs work did not create runtime, product, device,
  payment, pre/prod, daemon, adoption, or SEO proof.
- `blocked`: external comprehension, adoption, search ranking, and promotion
  impact require publication/indexing and later observation.

Primary source docs read:

- `docs/case-studies/tastyfuture-orchestration.md`
- `docs/reviews/2026-06-11-tastyfuture-orchestration-feedback.md`
- `docs/research/loop-engineering-alignment.md`
- `docs/research/agentic-engineering-feature-notes.md`
- `README.md`
- `README.zh-CN.md`

## Boundaries

Allowed paths used:

- `README.md`
- `README.zh-CN.md`
- `docs/**`

Forbidden paths not edited:

- `cmd/**`
- `.github/**`
- `Formula/**`
- `dist/**`
- release tags/notes
- package-manager distribution files
- credentials/secrets
- unrelated project files

## Summary

The README first screen now states the product in a faster GitHub-reader shape:
Codex App-first Loop Engineering, not a daemon, not package-manager first, not
a full agent OS, and not an unreviewed autonomous bot. It foregrounds task
contracts, isolated Codex worktree sessions, durable ledger/heartbeat status,
review/merge/cleanup discipline, evidence labels, stale-task rescue, and the
continuation guard.

The Chinese README mirrors the same positioning in natural Chinese rather than
a sentence-by-sentence stiff translation.

The new article explains the progression from ad-hoc AI coding chats to a
supervised engineering loop through the TastyFuture case: bounded contracts,
worktree isolation, repo-truth heartbeat checks, completed-unreviewed review
state, reviewer-owned merge/cleanup, practical workflow steps, evidence labels,
and explicit limits.

## Verification

Required checks:

- `git diff --check`
- simple link/path sanity check for newly added docs links

Code tests were not run because this change is docs-only and did not touch
source, build, release, package-manager, or routine implementation paths.

## Self-Review

- Diff reread: README opening diff and new article/review docs were reread
  before final checks.
- Allowed/forbidden paths: only `README.md`, `README.zh-CN.md`, and `docs/**`
  were changed.
- Docs consistency: README links point to existing repo-local docs, and the
  article's source trail points to existing case/review/research docs.
- Verification gaps: no runtime/product/device/prod proof is expected from
  this docs-only task.
- Residual risk: article quality and external comprehension need publication
  and reader feedback to validate.
