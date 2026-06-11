# Policy Auditor Negation And Human-Review Eval

Date: 2026-06-11

Scope:

- `eval/orchestration-policy-auditor/`
- `docs/roadmap.md`
- `docs/reviews/2026-06-11-policy-auditor-negation-human-review-eval.md`

Change:

Added the next bounded local/static orchestration-policy-auditor eval slice for
negation and human-review transcript semantics:

- `negated-policy-warnings-no-hit.json` keeps warning/rejection wording from
  becoming false positives when it mentions bad orchestration patterns as things
  that must not happen.
- `human-review-dry-run-dispatch-transcript.json` ensures a human-review
  transcript still trips `OPA001` when dry-run worker dispatch happened without
  a human gate.
- `human-review-main-checkout-fallback-transcript.json` ensures a human-review
  transcript still trips `OPA002` when setup failure falls back into the main
  checkout.
- `human-review-evidence-promotion-transcript.json` ensures a human-review
  transcript still trips `OPA005` when local/static evidence is promoted to
  direct proof.

Evidence labels:

- `local`: fixtures are deterministic repo-local JSON inputs under
  `eval/orchestration-policy-auditor/`.
- `local`: `go test ./...` passed.
- `local`: `go run ./cmd/codex-orchestrator eval run --repo . --json` passed
  with 16 deterministic fixtures after adding the new cases.
- `local`: `go run ./cmd/codex-orchestrator policy check --repo . --json`
  passed, scanning 43 repo-local orchestration policy input files with no rule
  hits and passing the 16 fixture eval cases.
- `local`: `git diff --check` passed.
- `blocked`: this slice does not parse private Codex App transcripts or prove
  runtime orchestration behavior.
- `blocked`: this slice does not add new OPA rules; it only extends regression
  coverage for the current local/static rule set.

Gates:

- `go test ./...` - passed.
- `go run ./cmd/codex-orchestrator eval run --repo . --json` - passed.
- `go run ./cmd/codex-orchestrator policy check --repo . --json` - passed.
- `git diff --check` - passed.

Self-review notes:

- Diff boundary stays inside the requested eval/docs paths.
- No `cmd/codex-orchestrator`, release, package-manager, Homebrew, npm, push,
  merge, or worktree cleanup path was touched.
- The new negation fixture separates explicit warnings/rejections from actual
  allowed/taken actions.
- The new human-review fixtures encode local reconstructed review text, not
  direct transcript ingestion.
- Residual risk: deeper discourse semantics remain heuristic because the
  current evaluator is a static string-rule suite.
