# Public Case Anonymization

Date: 2026-06-13

## Summary

Removed the real project name from public-facing codex-orchestrator docs and
sanitized the case study as a generic restaurant POS rewrite example.

## Changed

- Renamed the case study and article files from project-specific names to
  generic `restaurant-pos-*` names.
- Updated README, full guide, roadmap, article, case-study, and review links to
  use the anonymized file names.
- Replaced project-specific public prose with `restaurant POS rewrite` wording.
- Removed the project-specific default roadmap source from the helper in favor
  of a generic `docs/整体开发计划与进度.md` fallback. Real projects can still pass a
  custom roadmap scorer config.
- Removed a project-specific delegated-worker prompt phrase from the policy
  auditor trigger list.

## Evidence

- `local`: no public docs or helper source should contain the removed project
  name after this change.
- `blocked`: this is a docs/helper anonymization pass only. It does not claim
  runtime, production, adoption, SEO, or external index proof.
