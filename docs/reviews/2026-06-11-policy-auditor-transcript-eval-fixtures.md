# Policy Auditor Transcript Eval Fixtures

Date: 2026-06-11

Scope:

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

Evidence labels:

- `local`: fixtures are deterministic repo-local JSON inputs under
  `eval/orchestration-policy-auditor/`.
- `local`: source wording is reconstructed from repo-local review/roadmap
  failure descriptions, not from private transcript contents.
- `blocked`: this slice does not prove Codex App runtime behavior, scheduling
  delivery, or full V4 completion.

Expected verification:

- `go test ./...`
- `go run ./cmd/codex-orchestrator eval run --repo . --json`
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `git diff --check`

Self-review notes:

- No command implementation was changed.
- No release, package-manager, Homebrew, npm, tag, push, merge, or worktree
  cleanup path was touched.
- The fixtures exercise existing `OPA006` and `OPA007` rules only; deeper
  transcript semantics remain future work.
