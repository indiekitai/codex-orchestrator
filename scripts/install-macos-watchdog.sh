#!/usr/bin/env sh
set -eu

# Install a user LaunchAgent that runs macos-watchdog-run.sh periodically.
#
# Usage:
#   REPO=/path/to/project ./scripts/install-macos-watchdog.sh
#
# Optional env:
#   BIN=/Users/me/.local/bin/codex-orchestrator
#   INTERVAL=20m
#   MISSED_AFTER=45m
#   START_INTERVAL_SECONDS=1200
#   LABEL_SUFFIX=my-project
#   NOTIFY=1
#   SAY=0

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO=${REPO:-$(pwd)}
BIN=${BIN:-}
INTERVAL=${INTERVAL:-20m}
MISSED_AFTER=${MISSED_AFTER:-45m}
START_INTERVAL_SECONDS=${START_INTERVAL_SECONDS:-1200}
NOTIFY=${NOTIFY:-1}
SAY=${SAY:-0}

if [ -z "$BIN" ]; then
  if command -v codex-orchestrator >/dev/null 2>&1; then
    BIN=$(command -v codex-orchestrator)
  elif [ -x "$HOME/.local/bin/codex-orchestrator" ]; then
    BIN="$HOME/.local/bin/codex-orchestrator"
  else
    echo "codex-orchestrator binary not found. Set BIN=/path/to/codex-orchestrator." >&2
    exit 2
  fi
fi

if [ ! -x "$BIN" ]; then
  echo "BIN is not executable: $BIN" >&2
  exit 2
fi

if [ ! -d "$REPO/.git" ]; then
  echo "REPO does not look like a git checkout: $REPO" >&2
  exit 2
fi

if ! command -v launchctl >/dev/null 2>&1; then
  echo "launchctl is required on macOS" >&2
  exit 2
fi

hash=$(printf '%s' "$REPO" | cksum | awk '{print $1}')
repo_name=$(basename "$REPO" | tr -c '[:alnum:]' '-')
suffix=${LABEL_SUFFIX:-"$repo_name-$hash"}
label="com.indiekitai.codex-orchestrator.watchdog.$suffix"
agent_dir="$HOME/Library/LaunchAgents"
plist="$agent_dir/$label.plist"
stdout_log="$REPO/.codex-orchestrator/launchd-watchdog.out.log"
stderr_log="$REPO/.codex-orchestrator/launchd-watchdog.err.log"

mkdir -p "$agent_dir" "$REPO/.codex-orchestrator"

xml_escape() {
  printf '%s' "$1" | sed \
    -e 's/&/\&amp;/g' \
    -e 's/</\&lt;/g' \
    -e 's/>/\&gt;/g' \
    -e 's/"/\&quot;/g' \
    -e "s/'/\&apos;/g"
}

script_path_xml=$(xml_escape "$SCRIPT_DIR/macos-watchdog-run.sh")
repo_xml=$(xml_escape "$REPO")
bin_xml=$(xml_escape "$BIN")
interval_xml=$(xml_escape "$INTERVAL")
missed_after_xml=$(xml_escape "$MISSED_AFTER")
repo_name_xml=$(xml_escape "$repo_name")
notify_xml=$(xml_escape "$NOTIFY")
say_xml=$(xml_escape "$SAY")
stdout_log_xml=$(xml_escape "$stdout_log")
stderr_log_xml=$(xml_escape "$stderr_log")

cat >"$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$label</string>
  <key>ProgramArguments</key>
  <array>
    <string>/bin/sh</string>
    <string>$script_path_xml</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>REPO</key>
    <string>$repo_xml</string>
    <key>BIN</key>
    <string>$bin_xml</string>
    <key>INTERVAL</key>
    <string>$interval_xml</string>
    <key>MISSED_AFTER</key>
    <string>$missed_after_xml</string>
    <key>LABEL</key>
    <string>$repo_name_xml</string>
    <key>NOTIFY</key>
    <string>$notify_xml</string>
    <key>SAY</key>
    <string>$say_xml</string>
  </dict>
  <key>StartInterval</key>
  <integer>$START_INTERVAL_SECONDS</integer>
  <key>RunAtLoad</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$stdout_log_xml</string>
  <key>StandardErrorPath</key>
  <string>$stderr_log_xml</string>
</dict>
</plist>
EOF

launchctl unload "$plist" >/dev/null 2>&1 || true
launchctl load -w "$plist"

echo "Installed LaunchAgent: $label"
echo "Plist: $plist"
echo "Repo: $REPO"
echo "Report: $REPO/.codex-orchestrator/watchdog-heartbeat-report.json"
echo "Summary: $REPO/.codex-orchestrator/watchdog-heartbeat-summary.md"
echo "Logs: $stdout_log / $stderr_log"
echo
echo "To uninstall:"
echo "  launchctl unload '$plist' && rm '$plist'"
