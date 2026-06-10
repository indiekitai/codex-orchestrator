# Beta Usability Package

This document is the external-user path from "interesting skill" to "I can try
this safely on my own repository." It intentionally focuses on usability and
trust, not new orchestration features.

## Beta Goal

`codex-orchestrator` is ready for a beta when a new user can:

1. Install the skill and optional helper.
2. Understand the App-first boundary.
3. Initialize a local ledger.
4. Record one worker task.
5. Run `observe`, `heartbeat`, and at least one read-only routine.
6. Understand what the helper will not do.
7. Decide whether the result is safe to use in their project.

The beta is not a claim of a fully autonomous agent runtime. It is a packaged
Codex App orchestration workflow with a conservative local helper.

## Quickstart For A New User

### 1. Install the skill

Clone the repository, then copy it into Codex skills:

```bash
git clone https://github.com/indiekitai/codex-orchestrator.git
cd codex-orchestrator
mkdir -p ~/.codex/skills
cp -R . ~/.codex/skills/delegated-session-orchestrator
```

Open Codex App and start a session in the repository you want to orchestrate.
Ask for:

```text
Use $delegated-session-orchestrator for this repository.
Before dispatching, inspect git status, worktrees, and the roadmap.
Create isolated worktree sessions for worker tasks.
Review completed branches before merge.
Label evidence as direct, proxy, local, or blocked.
```

### 2. Install the optional helper

The helper is a Go CLI. It does not replace Codex App dispatch; it stores local
state and produces reports the App orchestrator can read.

```bash
scripts/install.sh
codex-orchestrator --help
```

If Go is not installed, download a prebuilt binary from GitHub Releases and put
it on `PATH`.

### 3. Initialize a disposable test repository

Do the first trial on a disposable repo or feature branch:

```bash
mkdir /tmp/codex-orchestrator-demo
cd /tmp/codex-orchestrator-demo
git init
echo '# demo' > README.md
git add README.md
git commit -m 'init demo'
codex-orchestrator init
```

### 4. Record a fake worker task

This simulates the state after Codex App creates a worker session/worktree:

```bash
git worktree add /tmp/codex-orchestrator-demo-worker -b codex/demo-task
cd /tmp/codex-orchestrator-demo-worker
echo 'hello from worker' > feature.txt
git add feature.txt
git commit -m 'add demo feature'

cd /tmp/codex-orchestrator-demo
codex-orchestrator record-task \
  --id DEMO-FEATURE-LOCAL \
  --title "Demo feature" \
  --worktree /tmp/codex-orchestrator-demo-worker \
  --branch codex/demo-task \
  --allowed 'feature.txt' \
  --gate 'git diff --check HEAD~1..HEAD' \
  --evidence local
```

### 5. Observe and run read-only routines

```bash
codex-orchestrator observe --json
codex-orchestrator heartbeat \
  --count 1 \
  --write-report .codex-orchestrator/heartbeat-report.json \
  --write-summary .codex-orchestrator/heartbeat-summary.md
codex-orchestrator run-routine pr-reviewer \
  --task-id DEMO-FEATURE-LOCAL \
  --write-report /tmp/pr-reviewer-report.json
```

Expected outcome:

- `observe` identifies the worker task from ledger and git truth.
- heartbeat writes a local report.
- `pr-reviewer` produces a read-only local review report.
- Nothing is merged, pushed, tagged, cleaned, or dispatched by the helper.

### 6. Let Codex App own the mutating steps

After the helper reports a task is ready, the Codex App orchestrator should:

1. Read the worker diff.
2. Check allowed and forbidden paths.
3. Check self-review and evidence labels.
4. Run credible gates.
5. Merge only if the review passes.
6. Push only if that is normal for the project.
7. Clean the worktree and local branch.
8. Append ledger events for review, merge, reject, block, or cleanup.

## App-First Boundary

`codex-orchestrator` has two layers:

- `SKILL.md`: tells Codex App how to coordinate sessions.
- `cmd/codex-orchestrator`: reports local repo/ledger/routine facts.

The helper is intentionally read-mostly. It does not:

- create Codex App sessions,
- continue Codex threads,
- merge branches,
- push to remotes,
- delete worktrees,
- delete branches,
- approve production, payment, hardware, or direct evidence.

This boundary matters because it keeps the dangerous decisions inside the
reviewing App orchestrator, where the user can inspect the evidence.

## Real App Demo Checklist

Before calling a beta release credible, run one real Codex App demo:

1. Create a fresh Codex App orchestrator session in a disposable repository.
2. Ask it to use `$delegated-session-orchestrator`.
3. Dispatch one small worker into an isolated worktree.
4. Record the worker in the helper ledger.
5. Let the worker commit and self-review.
6. Run `observe` and `pr-reviewer`.
7. Have the orchestrator review, merge, push if safe, and clean the worktree.
8. Append events for review, merge, and cleanup.
9. Run final `observe --json`.
10. Record the result in `docs/reviews/`.

This is the proof missing from a helper-only smoke: real App session dispatch,
review, merge, push, and cleanup behavior.

## Beta Release Checklist

Use this checklist before `v0.3.0-beta.1`:

- `go test ./...`
- `go build -trimpath -ldflags='-s -w' -o /tmp/codex-orchestrator ./cmd/codex-orchestrator`
- `codex-orchestrator validate-routines --dir routines`
- `codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `codex-orchestrator run-routine release-verifier --tag v0.3.0-beta.1 --repo . --json` after release publication
- README quickstart works from a clean clone.
- Chinese README matches the English quickstart at the workflow level.
- `SKILL.md` is synced with the release.
- GitHub release assets exist for the supported OS/arch matrix.
- Known limitations are visible before users try the tool.

## Known Beta Limitations

- Codex App session creation is still App-provided, not helper-provided.
- The helper does not run as a daemon.
- There is no Homebrew tap or npm wrapper yet.
- Routine runners are conservative local/proxy checkers.
- The release verifier uses GitHub metadata as proxy evidence.
- Direct proof of production, payment, hardware, or real deployed runtime still
  requires project-specific evidence.
- The tool improves orchestration discipline; it does not remove the need for
  engineering review.

## Recommended Next Package

After the beta usability package, the next large package should be one of:

1. **Real Codex App demo proof**: record the missing App dispatch/merge/cleanup
   evidence from the checklist above.
2. **Distribution package**: Homebrew tap, shell completions, and release notes.
3. **Daemon prototype**: an opt-in read-only watcher that runs `observe` and
   writes reports, without creating sessions or mutating git.

Do not add more tiny routines just to increase the count. Add a routine only
when it removes a named beta blocker or proves a common real workflow.
