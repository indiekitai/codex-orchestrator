# Distribution Package

This document is the beta distribution path for external users. It keeps the
human-facing setup simple: Codex App remains the primary entrypoint, while the
Go helper can be installed from source, release assets, or a Homebrew formula.

## Status

Current package: `v0.3.0-beta.2`

Implemented:

- Cross-platform GitHub release assets for macOS, Linux, and Windows.
- `scripts/install.sh` source install path for users with Go.
- Shell completion generation for bash, zsh, and fish.
- Homebrew formula draft at `Formula/codex-orchestrator.rb` that builds from
  the release tag.
- Release verifier routine for local tag and proxy GitHub release metadata.

Not yet implemented:

- Dedicated `homebrew-tap` repository.
- npm wrapper.
- Background daemon.

## Recommended User Flow

New users should still start by pasting the README bootstrap prompt into Codex
App. Codex App should explain which installation path it wants to use and why.
The human should not need to learn the helper CLI before the dry run.

## Install From Release Asset

Download the asset for your OS and architecture from:

https://github.com/indiekitai/codex-orchestrator/releases/tag/v0.3.0-beta.2

Example for macOS arm64:

```bash
curl -L -o /tmp/codex-orchestrator.tar.gz \
  https://github.com/indiekitai/codex-orchestrator/releases/download/v0.3.0-beta.2/codex-orchestrator_darwin_arm64.tar.gz
mkdir -p /tmp/codex-orchestrator
tar -xzf /tmp/codex-orchestrator.tar.gz -C /tmp/codex-orchestrator
install -m 0755 /tmp/codex-orchestrator/codex-orchestrator_darwin_arm64 ~/.local/bin/codex-orchestrator
codex-orchestrator --help
```

## Install From Source

```bash
git clone https://github.com/indiekitai/codex-orchestrator.git
cd codex-orchestrator
scripts/install.sh
codex-orchestrator --help
```

Use `BIN_DIR` to install somewhere else:

```bash
BIN_DIR=/usr/local/bin scripts/install.sh
```

## Shell Completions

The helper can print shell completion scripts:

```bash
codex-orchestrator completion bash
codex-orchestrator completion zsh
codex-orchestrator completion fish
```

Example installation paths:

```bash
# bash
mkdir -p ~/.local/share/bash-completion/completions
codex-orchestrator completion bash > ~/.local/share/bash-completion/completions/codex-orchestrator

# zsh
mkdir -p ~/.zsh/completions
codex-orchestrator completion zsh > ~/.zsh/completions/_codex-orchestrator

# fish
mkdir -p ~/.config/fish/completions
codex-orchestrator completion fish > ~/.config/fish/completions/codex-orchestrator.fish
```

For zsh, ensure `~/.zsh/completions` is in `fpath` before `compinit`.

## Homebrew Formula Draft

This repository includes a formula draft:

```text
Formula/codex-orchestrator.rb
```

Users can test it directly from a checkout:

```bash
brew install ./Formula/codex-orchestrator.rb
codex-orchestrator --help
```

The formula builds from the release tag and installs completions from the built
helper. This is not yet a dedicated Homebrew tap. Publishing a tap should be a
separate repository operation so updates, bottle policy, and release cadence
are clear.

## Release Notes

The release workflow should populate GitHub Release notes from:

```text
docs/beta-release-notes-draft.md
```

If GitHub API editing is blocked, the release can still be valid when the tag,
workflow, prerelease flag, and assets verify. Record the release-body failure as
`blocked` evidence instead of treating it as a binary release failure.

## Verification

Before announcing a distribution package:

```bash
go test ./...
go build -trimpath -ldflags='-s -w' -o /tmp/codex-orchestrator ./cmd/codex-orchestrator
/tmp/codex-orchestrator completion bash >/tmp/codex-orchestrator.bash
/tmp/codex-orchestrator completion zsh >/tmp/_codex-orchestrator
/tmp/codex-orchestrator completion fish >/tmp/codex-orchestrator.fish
go run ./cmd/codex-orchestrator validate-routines --dir routines
go run ./cmd/codex-orchestrator run-routine release-verifier --tag v0.3.0-beta.2 --repo . --json
```

Evidence labels:

- `local`: source build, completion generation, formula syntax inspection, local
  tag inspection.
- `proxy`: GitHub Release metadata and asset names from `gh`.
- `blocked`: unavailable Homebrew tap, failed GitHub API edit, missing release
  metadata, or network/auth failures.

Do not claim direct production, daemon, deployed runtime, payment, hardware, or
Codex App session-launch proof from this distribution package.
