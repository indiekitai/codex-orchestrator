# Policy Auditor Transcript Eval Fixtures

Date: 2026-06-11

Scope:

- `cmd/codex-orchestrator/main.go`
- `cmd/codex-orchestrator/main_test.go`
- `eval/orchestration-policy-auditor/`
- `docs/reviews/2026-06-11-policy-auditor-transcript-eval-fixtures.md`
- `docs/roadmap.md`

Change:

Added a bounded local/static follow-up for orchestration-policy-auditor eval
coverage beyond the first named-rule fixtures. The new fixtures encode
transcript-style review-note wording without depending on private transcripts:

- `heartbeat-current-binding-review-note.json` covers an `OPA006` stale heartbeat
  automation copied forward with `targetThreadId: "current"`.
- `pending-setup-chat-only-review-note.json` covers an `OPA007` case with
  `pendingWorktreeId` values kept only in chat or heartbeat prompt state without
  durable ledger recording.
- `child-complete-without-queue-proof.json` covers an `OPA003` transcript-style
  case where a single child task completion is used to stop the broader loop and
  delete heartbeat without continuation proof.
- `worker-boundary-forbidden-paths-missing.json` covers an `OPA004`
  transcript-style worker prompt that includes isolation, no-subagent/Paseo,
  self-review, and no-merge/push boundaries but omits a forbidden-path boundary.

The worker-boundary fixture exposed a narrow deterministic rule gap: `OPA004`
checked isolation, no subagents/Paseo, self-review, and no merge/push, but did
not require forbidden paths. The command change is limited to requiring a
forbidden-path/files/directories term inside paragraphs already classified as
worker/delegation prompts.

Evidence labels:

- `local`: fixtures are deterministic repo-local JSON inputs under
  `eval/orchestration-policy-auditor/`.
- `local`: source wording is reconstructed from repo-local review/roadmap
  failure descriptions, not from private transcript contents.
- `local`: `go run ./cmd/codex-orchestrator eval run --repo . --json` passed
  with 12 deterministic fixtures after adding the new cases.
- `blocked`: this slice does not prove Codex App runtime behavior, scheduling
  delivery, or full V4 completion.
- `blocked`: a first completion fixture variant used explicit missing
  ledger/roadmap/queue wording and was treated as guarded by the current string
  heuristic. That broader negation-aware semantic case remains future work.

Expected verification:

- `go test ./...`
- `go run ./cmd/codex-orchestrator eval run --repo . --json`
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `git diff --check`

Self-review notes:

- Command implementation changed only for the documented `OPA004`
  forbidden-path boundary miss exposed by
  `worker-boundary-forbidden-paths-missing.json`.
- No release, package-manager, Homebrew, npm, tag, push, merge, or worktree
  cleanup path was touched.
- The fixtures exercise existing `OPA003`, `OPA006`, and `OPA007` coverage plus
  one narrow `OPA004` boundary improvement; deeper transcript semantics remain
  future work.
