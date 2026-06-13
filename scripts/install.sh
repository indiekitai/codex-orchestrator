#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
BIN_DIR=${BIN_DIR:-"$HOME/.local/bin"}
BIN_NAME=${BIN_NAME:-codex-orchestrator}

if ! command -v go >/dev/null 2>&1; then
  echo "go is required to build $BIN_NAME" >&2
  echo "Install Go or download a prebuilt release binary." >&2
  exit 1
fi

mkdir -p "$BIN_DIR"
VERSION=$(cd "$ROOT_DIR" && git describe --tags --always --dirty 2>/dev/null || echo dev)
go build -trimpath -ldflags="-s -w -X main.helperVersion=$VERSION" -o "$BIN_DIR/$BIN_NAME" "$ROOT_DIR/cmd/codex-orchestrator"

echo "Installed $BIN_DIR/$BIN_NAME"
echo "Make sure $BIN_DIR is on PATH."
