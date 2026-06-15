# Claude Orchestrator Backport Review

Date: 2026-06-15

Scope: public `codex-orchestrator` review hardening after reading the related
Claude Code build-orchestrator skill and public `claude-orchestrator` repo.

## What Changed

- Added reviewer finding counts to `review import` with `--p0`, `--p1`, `--p2`,
  `--p3`, and `--other-findings`.
- Added package-level `findingTracker` output to `pack acceptance`.
- Added package-level `integrationGate` output so multi-task packages cannot
  silently rely only on per-worker gates.
- Added roadmap scorer hints for shared resources such as routes, localization
  strings, protocol/API contracts, database schema, DI registries, and config.
- Added roadmap scorer hints for common P1 patterns: DTO/serialization drift,
  state-machine coverage, tenant/store filters, money/cents arithmetic,
  nullable external fields, and idempotency/unique constraints.
- Updated `SKILL.md` so delegated workers self-review these patterns before
  handoff.

## Evidence Labels

- direct: local git diff, unit tests, and helper command behavior in this repo.
- local/static: roadmap scoring, package acceptance, and imported reviewer
  finding counts.
- proxy: external model or human review text imported with `review import`.
- blocked: none for this package.

## Boundaries

These changes are review aids. They do not authorize merge, push, cleanup,
deploy, release, or direct runtime/device/provider proof. External reviewer
output remains proxy/advisory evidence.

## Self-Review

- Diff stays inside helper code, tests, skill docs, roadmap, and review docs.
- No package-manager distribution work or Homebrew/npm changes.
- No private project names were added.
- The new fields are additive JSON fields and should be backward compatible for
  existing ledgers.
