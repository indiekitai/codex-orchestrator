#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
CODEX_HOME=${CODEX_HOME:-"$HOME/.codex"}
SKILL_DIR=${SKILL_DIR:-"$CODEX_HOME/skills/codex-orchestrator"}
BIN_DIR=${BIN_DIR:-"$HOME/.local/bin"}
BIN_NAME=${BIN_NAME:-codex-orchestrator}
INSTALL_HELPER=${INSTALL_HELPER:-auto}

usage() {
  cat >&2 <<'EOF'
usage: scripts/update-local.sh [--skill-only|--helper-only|--with-helper|--no-helper]

Updates the local Codex App skill from this repository checkout and optionally
rebuilds the Go helper. It does not run git pull, mutate project ledgers,
dispatch sessions, merge, push, or clean worktrees.

Environment:
  CODEX_HOME       default: ~/.codex
  SKILL_DIR        default: $CODEX_HOME/skills/codex-orchestrator
  BIN_DIR          default: ~/.local/bin
  BIN_NAME         default: codex-orchestrator
  INSTALL_HELPER   auto|always|never, default: auto
EOF
}

MODE=all
while [ "$#" -gt 0 ]; do
  case "$1" in
    --skill-only)
      MODE=skill
      ;;
    --helper-only)
      MODE=helper
      INSTALL_HELPER=always
      ;;
    --with-helper)
      INSTALL_HELPER=always
      ;;
    --no-helper)
      INSTALL_HELPER=never
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
  shift
done

copy_skill() {
  mkdir -p "$(dirname "$SKILL_DIR")"
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete \
      --exclude='.git' \
      --exclude='/.codex-orchestrator' \
      --exclude='/dist' \
      --exclude='/codex-orchestrator' \
      "$ROOT_DIR/" "$SKILL_DIR/"
  else
    rm -rf "$SKILL_DIR"
    mkdir -p "$SKILL_DIR"
    (cd "$ROOT_DIR" && tar \
      --exclude='./.git' \
      --exclude='./.codex-orchestrator' \
      --exclude='./dist' \
      --exclude='./codex-orchestrator' \
      -cf - .) | (cd "$SKILL_DIR" && tar -xf -)
  fi
  echo "Updated Codex skill: $SKILL_DIR"
}

install_helper() {
  if ! command -v go >/dev/null 2>&1; then
    echo "Skipped helper rebuild: go is not installed." >&2
    echo "Download a release binary or rerun after installing Go." >&2
    return 0
  fi
  mkdir -p "$BIN_DIR"
  go build -trimpath -ldflags="-s -w" -o "$BIN_DIR/$BIN_NAME" "$ROOT_DIR/cmd/codex-orchestrator"
  echo "Updated helper: $BIN_DIR/$BIN_NAME"
}

case "$MODE" in
  all)
    copy_skill
    if [ "$INSTALL_HELPER" = "always" ]; then
      install_helper
    elif [ "$INSTALL_HELPER" = "auto" ] && [ -x "$BIN_DIR/$BIN_NAME" ]; then
      install_helper
    else
      echo "Skipped helper rebuild. Use --with-helper to build it."
    fi
    ;;
  skill)
    copy_skill
    ;;
  helper)
    install_helper
    ;;
esac

if [ -x "$BIN_DIR/$BIN_NAME" ]; then
  "$BIN_DIR/$BIN_NAME" --help >/dev/null
  echo "Helper smoke passed: $BIN_DIR/$BIN_NAME --help"
fi
