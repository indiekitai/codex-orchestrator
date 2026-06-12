# Update UX Issue #1

Date: 2026-06-13

Scope: GitHub issue #1 feedback about how users should update
`codex-orchestrator` as the project evolves from a single skill file into a
skill plus helper/routines workflow.

## Change Summary

- Added `scripts/update-local.sh` for local checkout updates.
- Documented a Codex App-first update prompt in `README.md` and
  `README.zh-CN.md`.
- Added a command-line update path to `docs/v2-usage.md`.
- Updated `docs/CODEBASE_MAP.md` to include update scripts in the script
  ownership map.

## Evidence

- `local`: `scripts/update-local.sh` syncs the installed Codex skill from a
  checked-out repository and can rebuild the Go helper.
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

- The script updates from the current checkout. Users still need to fetch,
  pull, or download the repository version they want before running it.
- The helper has no dedicated `version` command yet; update smoke currently
  checks `codex-orchestrator --help`.
