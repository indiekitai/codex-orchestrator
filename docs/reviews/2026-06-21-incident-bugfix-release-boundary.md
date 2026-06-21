# Incident Bugfix Release Boundary

Date: 2026-06-21
Scope: codex-orchestrator incident and production-like bugfix rules

## Source Insight

A software-engineering-oriented Agent discussion emphasized that automated
online bug repair is less mature than AI-assisted prevention, reproduction, and
review. The useful rule for this repository is to keep Agent loops on the safe
side of the release boundary unless a project grants explicit incident
authority.

## Applied Rule

For incident or production-like bugfix packages, the default AI loop may:

- collect structured logs and relevant context;
- reconstruct reproduction steps;
- add a failing test or fixture when practical;
- prepare a bounded fix branch;
- run regression gates and produce review evidence.

It may not, by default:

- mutate production data;
- deploy or roll back;
- toggle provider, payment, or external-service behavior;
- run destructive remediation;
- claim direct production proof from local or proxy evidence.

Those actions require explicit project authorization for the exact action.

## Boundary

This is a rules and docs update only. It does not add a daemon, incident
runtime, production integration, deploy automation, helper behavior, or direct
proof.

## Docs Drift

Updated:

- `SKILL.md`
- `docs/full-guide.md`
- `docs/full-guide.zh-CN.md`
- `docs/roadmap.md`
