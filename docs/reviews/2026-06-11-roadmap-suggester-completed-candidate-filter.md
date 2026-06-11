# Roadmap Suggester Completed Candidate Filter

Date: 2026-06-11

Scope:

- `cmd/codex-orchestrator/main.go`
- `cmd/codex-orchestrator/main_test.go`
- `docs/reviews/2026-06-11-roadmap-suggester-completed-candidate-filter.md`

Change:

Added a conservative local/static filter to
`roadmap-next-task-suggester` so roadmap candidates whose own line clearly says
the work is already completed, done, covered, implemented, or shipped are
skipped before suggestion selection.

Regression coverage:

- Added a fixture where the roadmap still contains the completed
  `orchestration policy auditor follow-on eval fixtures` wording with Chinese
  `已补...` status text.
- Added an English `already completed` candidate to cover the same class of
  drift.
- Kept `budget policy report runner` in the same fixture and asserted it
  remains the primary local suggestion.

Evidence labels:

- `local`: deterministic Go unit test coverage in
  `TestRunRoadmapNextTaskSuggesterRoutineSkipsCompletedCandidateWording`.
- `local`: current repo run of `roadmap-next-task-suggester` skips the completed
  policy-auditor eval candidate and suggests `budget policy report runner`.
- `blocked`: this does not prove any Codex App heartbeat dispatch behavior; it
  only fixes the repo-local static routine output that the heartbeat loop can
  consume.

Gates:

- `go test ./cmd/codex-orchestrator -run 'TestRunRoadmapNextTaskSuggesterRoutine' -count=1` - passed.
- `go test ./...` - passed.
- `go run ./cmd/codex-orchestrator run-routine roadmap-next-task-suggester --repo . --json` - passed; completed policy-auditor eval wording was skipped and `budget policy report runner` remained suggested.
- `go run ./cmd/codex-orchestrator policy check --repo . --json` - passed.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json` - passed.

Residual risks:

- The filter is intentionally heuristic and conservative. It recognizes clear
  completion wording on the candidate line, status segment, or parenthetical
  note; ambiguous prose without completion markers may still need roadmap
  cleanup.
- The current parser still treats multiline roadmap bullets as line-based
  candidates, so long candidate descriptions may be shortened to the first line.
  This change does not broaden parser scope because the reported bug only needs
  completed-candidate suppression.
