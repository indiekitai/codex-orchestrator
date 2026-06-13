# README Human-Facing Split

Date: 2026-06-13

Scope: root README information architecture.

## Change Summary

- Rewrote `README.md` as a concise human-facing project homepage.
- Rewrote `README.zh-CN.md` with the same short structure.
- Preserved the previous long README content as:
  - `docs/full-guide.md`
  - `docs/full-guide.zh-CN.md`
- Updated `docs/CODEBASE_MAP.md` so future edits keep the root README short and
  move deep workflow/reference material into the full guide.

## Evidence

- `local`: documentation-only restructure.
- `local`: no helper command behavior, release workflow, skill runtime rules,
  ledger schema, or orchestration policy changed.

## Boundaries

- No package-manager distribution route was added.
- No release, tag, GitHub issue, or automation state was changed.
- No project ledger, worker dispatch, merge, push, cleanup, deploy, or runtime
  proof was performed.

## Residual Risk

- The full guide intentionally duplicates older README material. Future changes
  should keep the root README concise and update the full guide only when deep
  reference behavior changes.
