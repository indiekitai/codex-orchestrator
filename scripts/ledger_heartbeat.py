#!/usr/bin/env python3
"""Codex-orchestrator v2 helper CLI.

This helper is intentionally conservative. It records a durable ledger and
compares that ledger with local git truth. It never creates Codex sessions,
merges, pushes, deletes branches, or cleans worktrees.
"""

from __future__ import annotations

import argparse
import json
import subprocess
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


DEFAULT_STATE_DIR = ".codex-orchestrator"
DEFAULT_LEDGER = f"{DEFAULT_STATE_DIR}/ledger.json"
DEFAULT_EVENTS = f"{DEFAULT_STATE_DIR}/events.jsonl"


@dataclass
class CommandResult:
    ok: bool
    stdout: str
    stderr: str


def now_iso() -> str:
    return datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")


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


def resolve_path(path: str | None, default: str) -> Path:
    return Path(path or default).expanduser()


def read_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text())


def write_json(path: Path, data: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n")


def append_event(events_path: Path, event: dict[str, Any]) -> None:
    events_path.parent.mkdir(parents=True, exist_ok=True)
    events_path.open("a", encoding="utf-8").write(json.dumps(event, ensure_ascii=False) + "\n")


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


def default_branch(repo: Path) -> str:
    branch = run_git(repo, "branch", "--show-current")
    if branch.ok and branch.stdout:
        return branch.stdout
    ref = run_git(repo, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
    if ref.ok and "/" in ref.stdout:
        return ref.stdout.split("/", 1)[1]
    return "main"


def head_commit(repo: Path) -> str:
    result = run_git(repo, "rev-parse", "HEAD")
    return result.stdout if result.ok else ""


def load_ledger(path: Path) -> dict[str, Any]:
    if not path.exists():
        raise SystemExit(f"Ledger not found: {path}. Run init first.")
    return read_json(path)


def save_ledger(path: Path, ledger: dict[str, Any]) -> None:
    ledger["updatedAt"] = now_iso()
    write_json(path, ledger)


def find_task(ledger: dict[str, Any], task_id: str) -> dict[str, Any] | None:
    for task in ledger.get("tasks", []):
        if task.get("id") == task_id:
            return task
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


def observe_ledger(ledger_path: Path) -> dict[str, Any]:
    ledger = load_ledger(ledger_path)
    observations = [inspect_task(task) for task in ledger.get("tasks", [])]
    return {
        "ledger": str(ledger_path),
        "version": ledger.get("version"),
        "projectRoot": ledger.get("projectRoot"),
        "defaultBranch": ledger.get("defaultBranch"),
        "observedAt": now_iso(),
        "observations": observations,
    }


def print_observations(summary: dict[str, Any]) -> None:
    print(f"Ledger: {summary.get('ledger')}")
    print(f"Project: {summary.get('projectRoot')} default={summary.get('defaultBranch')}")
    for item in summary.get("observations", []):
        print()
        print(f"- {item.get('id')}: {item.get('status')}")
        print(f"  action: {item.get('action')}")
        print(f"  note: {item.get('note')}")
        if item.get("gitStatus"):
            print("  git:")
            for line in str(item["gitStatus"]).splitlines():
                print(f"    {line}")


def cmd_init(args: argparse.Namespace) -> int:
    ledger_path = resolve_path(args.ledger, DEFAULT_LEDGER)
    events_path = resolve_path(args.events, DEFAULT_EVENTS)
    repo = Path(args.project_root or ".").resolve()
    if ledger_path.exists() and not args.force:
        raise SystemExit(f"Ledger already exists: {ledger_path}. Use --force to overwrite.")

    created = now_iso()
    ledger = {
        "version": 1,
        "projectRoot": str(repo),
        "defaultBranch": args.default_branch or default_branch(repo),
        "remote": args.remote,
        "pushPolicy": args.push_policy,
        "maxConcurrency": args.max_concurrency,
        "createdAt": created,
        "updatedAt": created,
        "tasks": [],
    }
    write_json(ledger_path, ledger)
    append_event(events_path, {"at": created, "type": "init", "status": "created", "ledger": str(ledger_path)})
    print(f"Initialized ledger: {ledger_path}")
    print(f"Initialized events: {events_path}")
    return 0


def cmd_record_task(args: argparse.Namespace) -> int:
    ledger_path = resolve_path(args.ledger, DEFAULT_LEDGER)
    events_path = resolve_path(args.events, DEFAULT_EVENTS)
    ledger = load_ledger(ledger_path)

    if find_task(ledger, args.id):
        raise SystemExit(f"Task already exists: {args.id}")

    allowed = args.allowed or []
    forbidden = args.forbidden or []
    gates = args.gate or []
    task = {
        "id": args.id,
        "title": args.title or args.id,
        "threadId": args.thread_id,
        "worktree": args.worktree,
        "branch": args.branch,
        "baseCommit": args.base_commit or head_commit(Path(ledger.get("projectRoot", ".")).expanduser()),
        "status": args.status,
        "writeSet": {"allowed": allowed, "forbidden": forbidden},
        "gates": gates,
        "evidence": {"expected": args.evidence, "labels": ["direct", "proxy", "blocked"], "notes": args.evidence_note},
        "lastObservation": {"at": now_iso(), "result": args.status, "note": "Task recorded."},
        "history": [{"at": now_iso(), "type": "record-task", "status": args.status, "note": args.note or "Task recorded."}],
    }
    ledger.setdefault("tasks", []).append(task)
    save_ledger(ledger_path, ledger)
    append_event(events_path, {"at": now_iso(), "type": "record-task", "taskId": args.id, "status": args.status})
    print(f"Recorded task: {args.id}")
    return 0


def cmd_append_event(args: argparse.Namespace) -> int:
    ledger_path = resolve_path(args.ledger, DEFAULT_LEDGER)
    events_path = resolve_path(args.events, DEFAULT_EVENTS)
    ledger = load_ledger(ledger_path)
    task = find_task(ledger, args.task_id) if args.task_id else None
    event = {
        "at": now_iso(),
        "type": args.type,
        "status": args.status,
        "taskId": args.task_id,
        "note": args.note,
    }
    append_event(events_path, event)
    if task:
        task["status"] = args.status or task.get("status")
        task["lastObservation"] = {"at": event["at"], "result": task["status"], "note": args.note}
        task.setdefault("history", []).append({k: v for k, v in event.items() if v is not None})
        save_ledger(ledger_path, ledger)
    print(f"Appended event: {args.type}")
    return 0


def cmd_observe(args: argparse.Namespace) -> int:
    ledger_path = resolve_path(args.ledger, DEFAULT_LEDGER)
    summary = observe_ledger(ledger_path)
    write_report = getattr(args, "write_report", None)
    if write_report:
        write_json(Path(write_report).expanduser(), summary)
    if args.json:
        print(json.dumps(summary, indent=2, ensure_ascii=False))
    else:
        print_observations(summary)
    return 0


def cmd_status(args: argparse.Namespace) -> int:
    ledger_path = resolve_path(args.ledger, DEFAULT_LEDGER)
    ledger = load_ledger(ledger_path)
    counts: dict[str, int] = {}
    for task in ledger.get("tasks", []):
        status = task.get("status", "unknown")
        counts[status] = counts.get(status, 0) + 1
    result = {
        "ledger": str(ledger_path),
        "projectRoot": ledger.get("projectRoot"),
        "defaultBranch": ledger.get("defaultBranch"),
        "taskCount": len(ledger.get("tasks", [])),
        "counts": counts,
    }
    if args.json:
        print(json.dumps(result, indent=2, ensure_ascii=False))
    else:
        print(f"Ledger: {result['ledger']}")
        print(f"Project: {result['projectRoot']} default={result['defaultBranch']}")
        print(f"Tasks: {result['taskCount']}")
        for status, count in sorted(counts.items()):
            print(f"- {status}: {count}")
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Codex-orchestrator v2 helper CLI")
    subparsers = parser.add_subparsers(dest="command")

    init = subparsers.add_parser("init", help="Initialize a project-local ledger")
    init.add_argument("--ledger", help=f"Ledger path, default {DEFAULT_LEDGER}")
    init.add_argument("--events", help=f"Events path, default {DEFAULT_EVENTS}")
    init.add_argument("--project-root", help="Project root, default current directory")
    init.add_argument("--default-branch", help="Default integration branch")
    init.add_argument("--remote", default="origin")
    init.add_argument("--push-policy", default="manual")
    init.add_argument("--max-concurrency", type=int, default=2)
    init.add_argument("--force", action="store_true")
    init.set_defaults(func=cmd_init)

    record = subparsers.add_parser("record-task", help="Record a delegated task")
    add_common_paths(record)
    record.add_argument("--id", required=True)
    record.add_argument("--title")
    record.add_argument("--thread-id")
    record.add_argument("--worktree", required=True)
    record.add_argument("--branch", required=True)
    record.add_argument("--base-commit")
    record.add_argument("--status", default="active")
    record.add_argument("--allowed", action="append")
    record.add_argument("--forbidden", action="append")
    record.add_argument("--gate", action="append")
    record.add_argument("--evidence", default="local")
    record.add_argument("--evidence-note", default="")
    record.add_argument("--note")
    record.set_defaults(func=cmd_record_task)

    event = subparsers.add_parser("append-event", help="Append an event and optionally update one task")
    add_common_paths(event)
    event.add_argument("--task-id")
    event.add_argument("--type", required=True)
    event.add_argument("--status")
    event.add_argument("--note", default="")
    event.set_defaults(func=cmd_append_event)

    observe = subparsers.add_parser("observe", help="Read-only heartbeat observation")
    observe.add_argument("--ledger", help=f"Ledger path, default {DEFAULT_LEDGER}")
    observe.add_argument("--json", action="store_true")
    observe.add_argument("--write-report", help="Optional path to write JSON report")
    observe.set_defaults(func=cmd_observe)

    status = subparsers.add_parser("status", help="Summarize ledger task states")
    status.add_argument("--ledger", help=f"Ledger path, default {DEFAULT_LEDGER}")
    status.add_argument("--json", action="store_true")
    status.set_defaults(func=cmd_status)

    parser.add_argument("--ledger", help=argparse.SUPPRESS)
    parser.add_argument("--json", action="store_true", help=argparse.SUPPRESS)
    return parser


def add_common_paths(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--ledger", help=f"Ledger path, default {DEFAULT_LEDGER}")
    parser.add_argument("--events", help=f"Events path, default {DEFAULT_EVENTS}")


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()

    # Backward compatibility with the original helper:
    # scripts/ledger_heartbeat.py --ledger examples/ledger.example.json --json
    if args.command is None and args.ledger:
        return cmd_observe(args)

    if args.command is None:
        parser.print_help()
        return 2
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
