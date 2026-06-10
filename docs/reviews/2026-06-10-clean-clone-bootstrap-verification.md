# Clean Clone Bootstrap Verification

Date: 2026-06-10
Scope: `codex-orchestrator` beta bootstrap entry and clean-clone helper path
Source repository: `https://github.com/indiekitai/codex-orchestrator`
Clean clone path: `/tmp/codex-orchestrator-clean-verify`
Temporary skill home: `/tmp/codex-orchestrator-clean-home`
Temporary helper bin: `/tmp/codex-orchestrator-clean-bin`
Disposable demo repository: `/tmp/codex-orchestrator-clean-demo`

## Result

The clean-clone bootstrap path works locally:

1. A fresh clone from GitHub succeeded.
2. The README bootstrap entry was present in the cloned repository.
3. The skill copy path worked into a temporary Codex home:
   `/tmp/codex-orchestrator-clean-home/.codex/skills/delegated-session-orchestrator`.
4. `scripts/install.sh` built the Go helper into a temporary bin directory.
5. The helper command surface was available with `codex-orchestrator --help`.
6. Routine specs validated from the clean clone.
7. `docs-drift-checker` passed from the clean clone.
8. `evidence-label-auditor` passed from the clean clone.
9. The disposable demo flow from `docs/beta-usability-package.md` worked:
   `init`, worktree creation, worker commit, `record-task`, `observe`,
   `heartbeat`, and `pr-reviewer`.

This is local proof that the new "give Codex App a bootstrap prompt" entry is
backed by a runnable clean-clone setup path. It does not prove that every user
environment has compatible Codex App worktree/session tooling.

## Evidence

### Direct

None. This verification did not exercise production, daemon, deployed runtime,
hardware, payment, or external direct proof.

### Proxy

None used as success proof.

### Local

- `git clone https://github.com/indiekitai/codex-orchestrator.git
  /tmp/codex-orchestrator-clean-verify` succeeded.
- Clean clone `HEAD` was `2f92ec1`.
- Clean clone `git status --short --branch` reported `## main...origin/main`.
- Skill copy succeeded:
  `/tmp/codex-orchestrator-clean-home/.codex/skills/delegated-session-orchestrator/SKILL.md`
  existed after copy.
- `BIN_DIR=/tmp/codex-orchestrator-clean-bin ./scripts/install.sh` succeeded.
- `/tmp/codex-orchestrator-clean-bin/codex-orchestrator --help` printed the
  expected helper command surface.
- `/tmp/codex-orchestrator-clean-bin/codex-orchestrator validate-routines --dir routines`
  passed for all routine specs.
- `/tmp/codex-orchestrator-clean-bin/codex-orchestrator run-routine docs-drift-checker --repo . --json`
  passed.
- `/tmp/codex-orchestrator-clean-bin/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
  passed with no rule hits.
- Disposable demo repository initialized successfully.
- Demo worker worktree `/tmp/codex-orchestrator-clean-demo-worker` was created
  on `codex/demo-task`.
- Demo worker commit `c3a27b5` added `feature.txt`.
- `record-task` wrote `DEMO-FEATURE-LOCAL`.
- `observe --json` classified the demo task as `completed-unreviewed` and
  `overallStatus=review-needed`.
- `heartbeat --count 1` wrote both:
  - `.codex-orchestrator/heartbeat-report.json`
  - `.codex-orchestrator/heartbeat-summary.md`
- `run-routine pr-reviewer --task-id DEMO-FEATURE-LOCAL` passed and recorded:
  - task exists in ledger,
  - worktree exists,
  - branch matches,
  - worktree clean,
  - one commit after base,
  - committed diff adds `feature.txt`,
  - `git diff --check` passed.

### Blocked / Not Proven

- This was not a real Codex App session dispatch. That proof is covered by
  `docs/reviews/2026-06-10-real-codex-app-demo-proof.md`.
- This did not install into the user's real `~/.codex` or `~/.local/bin`; it
  used temporary directories to avoid mutating the user's live setup.
- This did not publish or verify a beta release tag.
- This did not exercise Homebrew, npm, shell completion, or daemon behavior.

## Commands

Clean clone and setup:

```bash
rm -rf /tmp/codex-orchestrator-clean-verify \
  /tmp/codex-orchestrator-clean-home \
  /tmp/codex-orchestrator-clean-bin
git clone https://github.com/indiekitai/codex-orchestrator.git \
  /tmp/codex-orchestrator-clean-verify
cd /tmp/codex-orchestrator-clean-verify
git rev-parse --short HEAD
git status --short --branch
mkdir -p /tmp/codex-orchestrator-clean-home/.codex/skills
cp -R . /tmp/codex-orchestrator-clean-home/.codex/skills/delegated-session-orchestrator
test -f /tmp/codex-orchestrator-clean-home/.codex/skills/delegated-session-orchestrator/SKILL.md
BIN_DIR=/tmp/codex-orchestrator-clean-bin ./scripts/install.sh
/tmp/codex-orchestrator-clean-bin/codex-orchestrator --help
/tmp/codex-orchestrator-clean-bin/codex-orchestrator validate-routines --dir routines
/tmp/codex-orchestrator-clean-bin/codex-orchestrator run-routine docs-drift-checker --repo . --json
/tmp/codex-orchestrator-clean-bin/codex-orchestrator run-routine evidence-label-auditor --repo . --json
```

Disposable demo:

```bash
BIN=/tmp/codex-orchestrator-clean-bin/codex-orchestrator
mkdir /tmp/codex-orchestrator-clean-demo
cd /tmp/codex-orchestrator-clean-demo
git init
git config user.name 'Codex Clean Verify'
git config user.email 'codex-clean-verify@example.invalid'
echo '# demo' > README.md
git add README.md
git commit -m 'init demo'
$BIN init
git worktree add /tmp/codex-orchestrator-clean-demo-worker -b codex/demo-task
cd /tmp/codex-orchestrator-clean-demo-worker
echo 'hello from worker' > feature.txt
git add feature.txt
git commit -m 'add demo feature'
cd /tmp/codex-orchestrator-clean-demo
$BIN record-task \
  --id DEMO-FEATURE-LOCAL \
  --title 'Demo feature' \
  --worktree /tmp/codex-orchestrator-clean-demo-worker \
  --branch codex/demo-task \
  --allowed 'feature.txt' \
  --gate 'git diff --check HEAD~1..HEAD' \
  --evidence local
$BIN observe --json
$BIN heartbeat --count 1 \
  --write-report .codex-orchestrator/heartbeat-report.json \
  --write-summary .codex-orchestrator/heartbeat-summary.md
$BIN run-routine pr-reviewer \
  --task-id DEMO-FEATURE-LOCAL \
  --write-report /tmp/clean-clone-pr-reviewer-report.json \
  --json
```

## Self-Review

- Diff boundary: this verification adds only this review document.
- Evidence labels: all proof is labeled local; no direct/proxy production,
  runtime, hardware, payment, release, or daemon claim is made.
- User experience: the clean-clone path supports the current README strategy:
  Codex App can read the GitHub repository, copy the skill, build the helper if
  useful, run a dry demo, and explain mutating steps before acting.
- Residual risk: a real beta still needs a release tag and asset verification
  after publication.
