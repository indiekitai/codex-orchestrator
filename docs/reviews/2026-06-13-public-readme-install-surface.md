# Public README Install Surface

Date: 2026-06-13

## Summary

Tightened the public repository surface without publishing a new release.

## Changed

- Removed the stale Homebrew Formula from the repository root. Homebrew/tap
  distribution is not a supported product path for this project, and the
  Formula pointed at an old beta tag.
- Clarified the README Quick Start language so a new user knows to copy the
  prompt into Codex App from the target repository.
- Clarified that command-line `self-update` examples are for users who already
  have the helper installed.

## Evidence

- `local`: docs/install-surface cleanup only.
- `blocked`: no new release/tag was published in this pass.
