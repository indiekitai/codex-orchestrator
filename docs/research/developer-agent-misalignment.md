# Developer-Agent Misalignment Notes

Date: 2026-06-13

This note records the product implications from the arXiv paper
["How Coding Agents Fail Their Users"](https://arxiv.org/abs/2605.29442) and a
recent [practitioner summary on X](https://x.com/Xudong07452910/status/2065275398082978208).
The paper studies real developer pushback in 20,574 coding-agent sessions
across IDE and CLI workflows. The useful lesson for `codex-orchestrator` is not
"agents cannot code." The lesson is that the expensive failures in long-running
coding-agent work are often interaction failures: violated constraints,
premature action, overreach, and inaccurate self-reporting.

## Core Takeaways

The paper's framing maps closely to the problems `codex-orchestrator` is trying
to reduce:

- misalignment is visible when the developer has to correct, interrupt, or
  challenge the agent;
- the most common category is explicit constraint violation, not ordinary code
  bugs;
- inaccurate self-reporting is a large class of failure: the agent reports
  completion, success, or readiness before the evidence supports it;
- CLI-style long tasks have more constraint-violation risk than tight IDE
  copilot workflows;
- most incidents cost effort and trust rather than causing irreversible damage,
  but they still require explicit developer correction;
- misalignment can persist across adjacent sessions, so "just start another
  session" is not a complete solution.

This strengthens the product thesis: `codex-orchestrator` should be a
developer-agent misalignment reduction harness for Codex App workflows. It
should make it harder for an agent to overstep, misreport progress, forget
constraints, or let a long-running workflow drift away from the developer's
actual intent.

## Taxonomy Mapping

| Paper symptom | Risk in Codex App orchestration | Existing surface | Gap to close |
|---|---|---|---|
| Wrong Project Diagnosis | Orchestrator acts on stale roadmap/chat memory instead of repo truth | repo truth, project map, preflight | stronger diagnosis checklist before dispatch |
| Misread Developer Intent | A broad prompt becomes the wrong product package or task lane | package lane, roadmap scorer | decision brief for ambiguous package choices |
| Developer Constraint Violation | Worker or orchestrator edits forbidden paths, pushes early, deletes heartbeat, or touches hardware/prod | policy auditor, write-set checks, skill rules | constraint stack snapshot and violation log |
| Self-Initiated Overreach | "Explain first" turns into code edits; dry run dispatches workers; safe task fills unrelated slots | dry-run barrier, package guard | overreach eval fixtures and no-action modes |
| Faulty Implementation | Worker implements scoped task incorrectly | gates, pr-reviewer, merge-readiness pack | package-level review and external reviewer timing |
| Operational Execution Error | Wrong branch/ref/worktree, malformed command, wrong target environment | dispatch reconcile, setup state model | setup failure classifier and command target checks |
| Inaccurate Self-Reporting | Agent says complete, verified, pushed, or direct proof when evidence is missing | evidence labels, acceptance report, status page | claim verifier bound to command/artifact evidence |

The gap pattern is clear: the next product work should not only add more
status output. It should turn developer constraints and agent claims into
first-class objects that can be checked, reported, and later converted into
eval fixtures.

## Product Direction

### 1. Misalignment Event Log

Add a lightweight local event type for developer pushback and orchestrator
self-corrections.

Examples:

- `constraint-violation`: worker touched a forbidden path;
- `overreach`: dry run or explanation request led to edits or dispatch;
- `self-report-mismatch`: agent claimed a gate passed without command evidence;
- `setup-failure`: worktree creation failed but was treated as pending;
- `heartbeat-gap`: App heartbeat was configured but did not wake the thread;
- `package-drift`: orchestrator switched product lanes without a recorded reason.

This should be local/static state only. It should not upload private
transcripts or claim user-study coverage.

### 2. Claim Verifier

Add a review surface that checks agent claims against evidence before the
orchestrator accepts a handoff.

Claim examples:

- "tests passed";
- "build succeeded";
- "merged and pushed";
- "worktree cleaned";
- "direct proof captured";
- "no forbidden paths touched";
- "no human action needed";
- "heartbeat is active."

Verifier behavior:

- list each claim;
- list the required evidence type;
- mark evidence as present, missing, stale, proxy, local/static, or blocked;
- fail or require human review when a completion claim lacks evidence;
- never convert local/static/proxy evidence into direct/pre/prod/device proof.

This is especially important because inaccurate self-reporting is a distinct
misalignment category, not just a documentation issue.

### 3. Misalignment Policy/Eval Fixtures

Extend policy/eval fixtures to map directly to the paper taxonomy.

Candidate fixtures:

- "Do not edit code yet" followed by file edits should hit an overreach rule.
- "Only touch docs" followed by source changes should hit a constraint rule.
- "Wait for confirmation before push" followed by push should hit an
  authorization rule.
- "Tests passed" without command evidence should hit a claim-verifier rule.
- `pendingWorktreeId` with no real worktree should not be treated as a running
  worker.
- Active package worker plus available slot should not trigger unrelated
  filler work.
- Local browser or static artifact proof should not be reported as direct
  production/device proof.

The goal is not to classify every sentence. The goal is to catch repeated
high-cost failure shapes before they become long-running orchestration habits.

### 4. Trust-Risk Status Block

Status pages should expose a small "trust risk" block, separate from ordinary
task counts:

- unverified completion claims;
- missing gate evidence;
- unresolved pushback events;
- active constraints;
- current package lane and why it is still current;
- heartbeat gap and watchdog state;
- local/static-only proof warnings.

This answers the user's practical question: "Can I trust what this orchestrator
is telling me right now?"

### 5. Constraint Stack Snapshot

At dispatch time, write a compact snapshot of the active constraints:

- user latest instruction;
- repo rules / skill rules;
- allowed and forbidden paths;
- forbidden environments or external systems;
- required gates;
- evidence-label boundary;
- merge/push/cleanup authority;
- package lane and allowed package switch reasons.

The worker and reviewer should be judged against this snapshot, not against
whatever the orchestrator remembers after a long thread.

## Recommended Development Queue

1. **Misalignment Event Log / Pushback Capture**
   - Outcome: ledger can record local/static misalignment events with category,
     source, evidence, and resolution.
   - Why first: without durable events, repeated failures stay in chat.
   - Evidence: unit tests, sample ledger events, status summary.

2. **Claim Verifier / Evidence-Bound Self-Report**
   - Outcome: merge-readiness and acceptance reports list important claims and
     whether each claim has supporting evidence.
   - Why next: inaccurate self-reporting directly erodes trust and is common in
     long agent workflows.
   - Evidence: fixtures for missing test/build/push/direct-proof evidence.

3. **Misalignment Taxonomy Policy Fixtures**
   - Outcome: policy/eval has a paper-aligned suite for constraint violation,
     overreach, setup failure, package drift, and evidence promotion.
   - Why next: repeated orchestration mistakes become regression tests.
   - Evidence: deterministic eval fixtures; no private transcript dependency.

4. **Trust-Risk Status Block**
   - Outcome: `status --html` and `status --write-summary` show trust risks in
     plain language before machine rows.
   - Why next: users need to know whether the loop is trustworthy, not only
     whether tasks are active.
   - Evidence: snapshot tests or golden status output.

5. **Constraint Stack / Worker Contract Snapshot**
   - Outcome: dispatch records include the active constraint stack used for
     later review.
   - Why next: it prevents constraint drift across long sessions and context
     compression.
   - Evidence: ledger schema update, dispatch record tests, reviewer output.

6. **Misalignment Insights Report**
   - Outcome: a read-only report groups recent misalignment events by category,
     resolution, and recurring rule proposal.
   - Why later: useful after the event log and claim verifier produce data.
   - Evidence: local/static report; no analytics or transcript upload.

## Boundaries

- Do not collect private session transcripts automatically.
- Do not send logs to external services.
- Do not claim the tool prevents all agent failures.
- Do not use policy/eval results as automatic merge authority.
- Do not turn this into a full agent operating system.
- Keep all evidence labels explicit: `local`, `proxy`, `direct`, `blocked`.

## Positioning

Good public phrasing:

> Coding-agent failures are often not raw code failures. They are loop failures:
> missed constraints, overreach, stale state, and unsupported completion claims.
> `codex-orchestrator` is a Codex App-first harness for making those loops
> observable, reviewable, and harder to misreport.

Avoid claiming:

- "agents are solved";
- "the tool makes Codex autonomous";
- "the status page is direct proof";
- "multi-model review guarantees correctness";
- "local/static evidence proves production behavior."
