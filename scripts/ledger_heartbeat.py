#!/usr/bin/env python3
"""Read-only heartbeat checker for codex-orchestrator ledgers.

This helper is intentionally conservative. It compares a ledger with local git
truth and prints suggested orchestrator actions. It never edits files, merges,
pushes, deletes branches, or creates sessions.
"""

from __future__ import annotations

import argparse
import json
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Any


@dataclass
class CommandResult:
    ok: bool
    stdout: str
    stderr: str


def run_git(cwd: Path, *args: str) -> CommandResult:
    try:
        proc = subprocess.run(
            ["git", *args],
            cwd=cwd,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
    except FileNotFoundError as exc:
        return CommandResult(False, "", str(exc))
    return CommandResult(proc.returncode == 0, proc.stdout.strip(), proc.stderr.strip())


def current_branch(status_output: str) -> str | None:
    first = status_output.splitlines()[0] if status_output else ""
    if not first.startswith("## "):
        return None
    branch = first[3:].split("...", 1)[0].strip()
    return None if branch == "HEAD (no branch)" else branch


def has_dirty_changes(status_output: str) -> bool:
    return any(line and not line.startswith("## ") for line in status_output.splitlines())


def has_commits_after_base(worktree: Path, base_commit: str | None) -> bool | None:
    if not base_commit:
        return None
    if set(base_commit) == {"0"}:
        return None
    result = run_git(worktree, "rev-list", "--count", f"{base_commit}..HEAD")
    if not result.ok:
        return None
    try:
        return int(result.stdout or "0") > 0
    except ValueError:
        return None


def inspect_task(task: dict[str, Any]) -> dict[str, Any]:
    worktree_value = task.get("worktree")
    expected_branch = task.get("branch")
    base_commit = task.get("baseCommit")

    if not worktree_value:
        return {
            "id": task.get("id"),
            "status": "blocked",
            "action": "record missing worktree path",
            "note": "Task has no worktree path in ledger.",
        }

    worktree = Path(worktree_value).expanduser()
    if not worktree.exists():
        return {
            "id": task.get("id"),
            "status": "pending-setup",
            "action": "verify setup or mark stale if expired",
            "note": f"Worktree does not exist: {worktree}",
        }

    status = run_git(worktree, "status", "--short", "--branch")
    if not status.ok:
        return {
            "id": task.get("id"),
            "status": "blocked",
            "action": "inspect worktree git state",
            "note": status.stderr or "git status failed",
        }

    branch = current_branch(status.stdout)
    dirty = has_dirty_changes(status.stdout)
    commits_after_base = has_commits_after_base(worktree, base_commit)

    if expected_branch and branch and branch != expected_branch:
        return {
            "id": task.get("id"),
            "status": "blocked",
            "action": "fix branch mismatch before review",
            "note": f"Expected {expected_branch}, found {branch}.",
            "gitStatus": status.stdout,
        }

    if dirty:
        return {
            "id": task.get("id"),
            "status": "stale-needs-inspection",
            "action": "inspect uncommitted scoped diff or nudge same worker",
            "note": "Worktree has uncommitted changes.",
            "gitStatus": status.stdout,
        }

    if commits_after_base is True:
        return {
            "id": task.get("id"),
            "status": "completed-unreviewed",
            "action": "orchestrator review required before merge",
            "note": "Clean worktree has commits after baseCommit.",
            "gitStatus": status.stdout,
        }

    if commits_after_base is None:
        return {
            "id": task.get("id"),
            "status": task.get("status", "active"),
            "action": "inspect manually",
            "note": "Could not compare baseCommit; ledger may be a template or base is missing.",
            "gitStatus": status.stdout,
        }

    return {
        "id": task.get("id"),
        "status": "active",
        "action": "quiet",
        "note": "Clean worktree has no commits after baseCommit.",
        "gitStatus": status.stdout,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Read-only codex-orchestrator ledger heartbeat")
    parser.add_argument("--ledger", required=True, help="Path to ledger JSON")
    parser.add_argument("--json", action="store_true", help="Print JSON output")
    args = parser.parse_args()

    ledger_path = Path(args.ledger).expanduser()
    ledger = json.loads(ledger_path.read_text())

    observations = [inspect_task(task) for task in ledger.get("tasks", [])]
    summary = {
        "ledger": str(ledger_path),
        "version": ledger.get("version"),
        "projectRoot": ledger.get("projectRoot"),
        "defaultBranch": ledger.get("defaultBranch"),
        "observations": observations,
    }

    if args.json:
        print(json.dumps(summary, indent=2, ensure_ascii=False))
        return 0

    print(f"Ledger: {ledger_path}")
    print(f"Project: {ledger.get('projectRoot')} default={ledger.get('defaultBranch')}")
    for item in observations:
        print()
        print(f"- {item.get('id')}: {item.get('status')}")
        print(f"  action: {item.get('action')}")
        print(f"  note: {item.get('note')}")
        if item.get("gitStatus"):
            print("  git:")
            for line in str(item["gitStatus"]).splitlines():
                print(f"    {line}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

