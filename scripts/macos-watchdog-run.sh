#!/usr/bin/env sh
set -eu

# One-shot macOS watchdog runner for hands-off orchestration.
# Intended for launchd. It does not dispatch, merge, push, or clean worktrees.
# It uses heartbeat --check-only so the external watchdog does not append an App
# heartbeat event and mask a missed Codex App wakeup.

REPO=${REPO:-}
BIN=${BIN:-codex-orchestrator}
INTERVAL=${INTERVAL:-20m}
MISSED_AFTER=${MISSED_AFTER:-45m}
STATE_DIR=${STATE_DIR:-.codex-orchestrator}
LABEL=${LABEL:-codex-orchestrator}
NOTIFY=${NOTIFY:-1}
SAY=${SAY:-0}

if [ -z "$REPO" ]; then
  echo "REPO is required" >&2
  exit 2
fi

if [ ! -d "$REPO/.git" ]; then
  echo "REPO does not look like a git checkout: $REPO" >&2
  exit 2
fi

cd "$REPO"
mkdir -p "$STATE_DIR"

REPORT="$STATE_DIR/watchdog-heartbeat-report.json"
SUMMARY="$STATE_DIR/watchdog-heartbeat-summary.md"
STDOUT_LOG="$STATE_DIR/watchdog-last-stdout.json"
ERROR_LOG="$STATE_DIR/watchdog-last-error.log"

if ! "$BIN" heartbeat \
  --repo "$REPO" \
  --count 1 \
  --check-only \
  --interval "$INTERVAL" \
  --missed-after "$MISSED_AFTER" \
  --write-report "$REPORT" \
  --write-summary "$SUMMARY" \
  --json >"$STDOUT_LOG" 2>"$ERROR_LOG"; then
  message="codex-orchestrator watchdog failed for ${LABEL}. See $ERROR_LOG"
  echo "$message" >&2
  if [ "$NOTIFY" = "1" ] && command -v osascript >/dev/null 2>&1; then
    osascript -e "display notification \"$(printf '%s' "$message" | sed 's/"/\\"/g')\" with title \"codex-orchestrator\"" >/dev/null 2>&1 || true
  fi
  exit 1
fi

if grep -q '"heartbeatStatus"' "$REPORT" && grep -q '"status": "missed"' "$REPORT"; then
  message="可能漏跑了 codex-orchestrator heartbeat：${LABEL}。请查看 $SUMMARY"
  echo "$message"
  if [ "$NOTIFY" = "1" ] && command -v osascript >/dev/null 2>&1; then
    osascript -e "display notification \"$(printf '%s' "$message" | sed 's/"/\\"/g')\" with title \"codex-orchestrator\"" >/dev/null 2>&1 || true
  fi
  if [ "$SAY" = "1" ] && command -v say >/dev/null 2>&1; then
    say "$message" >/dev/null 2>&1 || true
  fi
fi
