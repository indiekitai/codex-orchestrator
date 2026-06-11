# App-First Install UX Review

Date: 2026-06-11
Scope: beta install positioning for `codex-orchestrator`

## Result

The install story is now explicitly Codex App-first:

1. A new user should start by pasting the README bootstrap prompt into Codex App.
2. Codex App reads the GitHub repository, installs the `codex-orchestrator`
   skill if needed, and explains the dry run.
3. The Go helper binary is optional and exists to support durable ledger,
   `observe`, heartbeat reports, and routine checks.
4. Homebrew is optional/later package-manager convenience, not a beta blocker
   and not the primary entrypoint.

## Evidence

### Local

- `README.md` and `README.zh-CN.md` now say users do not need to install a CLI
  or Homebrew tap before trying the workflow.
- `docs/beta-usability-package.md` now records the real App demo proof as
  complete and keeps the next package focused on App-first install UX.
- `docs/distribution-package.md` keeps release assets documented while stating
  that release binaries are helper installation paths, not the product's
  primary entrypoint.
- `docs/roadmap.md` now says Homebrew/npm should only be pursued if users
  explicitly want package-manager-managed helper binaries.

### Blocked / Not Claimed

- No new Homebrew tap was created.
- No npm wrapper was created.
- No daemon behavior was implemented or claimed.
- No new Codex App worker session was needed for this docs-only install UX
  clarification because the real App demo proof was already recorded in
  `docs/reviews/2026-06-10-real-codex-app-demo-proof.md`.

## Verdict

Accepted as an App-first install UX clarification. The next product step should
not be Homebrew by default; it should be either a clearer external demo, a
read-only watcher prototype, or user-driven install friction fixes discovered
from real use.
