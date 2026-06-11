# v0.3.2 Release Notes

`v0.3.2` strengthens the review and decision handoff layer around
`codex-orchestrator`. The release keeps the same Codex App-first workflow, but
makes local/static packs more decision-ready for humans, orchestrators, and
external reviewers.

## Highlights

- Added model-agnostic loop strategy notes:
  `docs/research/model-plateau-loop-engineering.md`.
- Updated the roadmap to prioritize portable review artifacts over
  model-specific prompt tricks.
- Strengthened `pack consultation` with:
  - `ownerDecisionBrief`
  - `authorizationMatrix`
  - `liveProofGate`
- Strengthened `pack merge-readiness` with:
  - `authorizationMatrix`
  - `liveProofGate`
  - `acceptanceReport`
- Updated `SKILL.md` so orchestrators separate review evidence, merge
  acceptance, push closeout, cleanup, release/deploy authorization, and live
  proof or waiver requirements.
- Added review documentation:
  `docs/reviews/2026-06-11-decision-brief-authorization-live-proof-pack.md`.

## Why This Release

Recent maintainer-orchestrator practice converges on the same lesson:
delegated work should not hand a human a vague blocker or a bare URL. A useful
orchestrator should prepare a decision-ready brief, name the evidence, separate
permissions, and make live-proof gaps explicit.

`v0.3.2` brings that discipline into the existing consultation and
merge-readiness packs without turning the helper into a daemon, release bot, or
GitHub-specific maintainer tool.

## New Pack Fields

`pack consultation` now includes:

- `ownerDecisionBrief`: what is blocked, why a decision is needed now, what
  proof exists, what is missing, available choices, tradeoffs, and the
  recommendation.
- `authorizationMatrix`: records that consultation only authorizes asking the
  owner, not implementation, merge, push, cleanup, or release.
- `liveProofGate`: records whether live/runtime/device/provider proof appears
  required, whether it exists, and whether a waiver would be needed.

`pack merge-readiness` now includes:

- `authorizationMatrix`: review evidence is not the same as merge, push,
  cleanup, or release authorization.
- `liveProofGate`: runtime/device/provider proof remains separate from local
  static review evidence.
- `acceptanceReport`: a draft review outcome such as `review-ready`,
  `needs-review`, `reject-for-fixup`, or `blocked`, with evidence and residual
  risks.

## Evidence Boundary

The new fields are still `local/static` review aids. They do not:

- create Codex App sessions,
- dispatch workers,
- merge or push,
- clean worktrees,
- edit ledger or git state,
- call the network,
- perform live runtime/device/provider checks,
- authorize release, deploy, tag, registry publish, production mutation, or
  external-service action.

Direct proof and item-specific waivers remain outside the pack until a human or
orchestrator records them explicitly.

## Verification Before Publishing

Checks used for this release:

- `go test ./cmd/codex-orchestrator -run 'TestPackConsultation|TestPackMergeReadiness'`
- `go test ./...`
- `go run ./cmd/codex-orchestrator policy check --repo .`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `git diff --check`

The local helper was rebuilt and the installed Codex skill was synced before
publishing.

## Suggested Announcement

`codex-orchestrator v0.3.2` adds decision-ready handoff fields to consultation
and merge-readiness packs: owner decision briefs, authorization matrices, live
proof gates, and acceptance report drafts. The goal is still App-first Loop
Engineering: isolate work, review evidence, keep permissions explicit, and
never confuse local/static proof with live/direct proof.
