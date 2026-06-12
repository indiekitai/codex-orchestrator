# Update UX Issue #1

Date: 2026-06-13

Scope: GitHub issue #1 feedback about how users should update
`codex-orchestrator` as the project evolves from a single skill file into a
skill plus helper/routines workflow.

## Change Summary

- Added `scripts/update-local.sh` for local checkout updates.
- Added `codex-orchestrator self-update` as the user-facing update command,
  including a `--from-github` path for users without a local checkout.
- Documented a Codex App-first update prompt in `README.md` and
  `README.zh-CN.md`.
- Added a command-line update path to `docs/v2-usage.md`.
- Updated `docs/CODEBASE_MAP.md` to include update scripts in the script
  ownership map.

## Evidence

- `local`: `scripts/update-local.sh` syncs the installed Codex skill from a
  checked-out repository and can rebuild the Go helper.
- `local`: `codex-orchestrator self-update` resolves a checkout or installed
  skill directory and delegates to `scripts/update-local.sh`.
- `local/proxy`: `codex-orchestrator self-update --from-github` can clone or
  fetch this repository into a local update cache before syncing the local
  install. GitHub fetch success is repository-source evidence, not proof that
  any project orchestration state is correct.
- `local`: the update path explicitly avoids project ledgers, dispatch,
  merge, push, release, and worktree cleanup.
- `local/static`: README and v2 usage docs now include a copy-paste update
  prompt for Codex App users.

## Boundaries

- No package-manager distribution route was added.
- No GitHub release, tag, release asset, Homebrew, npm, or tap workflow was
  changed.
- No project `.codex-orchestrator/ledger.json` state is modified by the update
  script.

## Residual Risk

- The helper has no dedicated `version` command yet; update smoke currently
  checks `codex-orchestrator --help`.
