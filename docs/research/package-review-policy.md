# Package Review Policy

Date: 2026-06-12

This note captures the product decision behind package-level external review in
codex-orchestrator. It is intentionally separate from the command reference so
future work can check the reasoning before changing the implementation.

## Problem

Codex App can finish many isolated worker tasks in sequence. If every small task
is reviewed by another model, the workflow becomes expensive and noisy. If no
external model ever reviews the accumulated feature package, the orchestrator can
miss cross-slice bugs, contract drift, evidence inflation, or shallow progress.

The right unit for external review is therefore usually the **feature package**:
several related workers that together form one product outcome.

## Product Rule

Do not run external review for every small worker by default.

Run external review at package boundaries when one of these is true:

- three to five related worker tasks form one feature package;
- the package will be reported as one user-facing outcome;
- the package touches shared contract, API envelope, DB migration, auth,
  security, payment, hardware, provider, pre/prod, or deployment boundaries;
- the orchestrator is about to mark a product module as complete, stable, or
  ready for stronger proof.

The external review output is always `proxy/advisory` evidence. It can block an
acceptance decision or inform a fixup, but it cannot by itself authorize
implementation, merge, push, cleanup, release, deploy, or direct runtime/device/
provider proof.

## Reviewer Selection

Default policy:

| Package risk | Action |
|---|---|
| Low: docs, copy, local shell, narrow UI polish | External review optional; generating a pack is enough. |
| Medium: normal feature package, 3-5 related local/proxy slices | Run one reviewer, defaulting to `pi`. |
| High: shared contract, DB migration, API envelope, auth/security, payment, hardware, provider, pre/prod | Run two reviewers, defaulting to `pi` and `claude`. |
| Reviewer unavailable | Generate the pack, mark review setup `blocked`, and allow manual import from DeepSeek, Claude, Pi, or a human reviewer. |

The built-in default runner order is:

1. `pi` as the primary local reviewer.
2. `claude` as the secondary high-risk reviewer.
3. `codex` as a possible same-family fallback, not an independent-model
   substitute.
4. `deepseek` and `human` as manual/import-only reviewers unless a stable local
   runner is configured later.

`claude ultrareview` is not part of the default workflow. If it is ever added, it
should be modeled as a separate high-cost policy option, not as the ordinary
package-review path.

## Configuration Shape

Repository-level policy should live at:

```text
.codex-orchestrator/review-policy.json
```

Suggested default:

```json
{
  "reviewPolicyVersion": 1,
  "defaultMode": "package-boundary",
  "primaryReviewer": "pi",
  "secondaryReviewer": "claude",
  "fallbackReviewers": ["codex"],
  "manualReviewers": ["deepseek", "human"],
  "trigger": {
    "minTasksInPackage": 3,
    "maxTasksBeforeReview": 5,
    "requireForRisk": [
      "shared-contract",
      "db-migration",
      "api-envelope",
      "auth-security",
      "payment",
      "hardware",
      "provider",
      "pre-prod"
    ]
  },
  "decision": {
    "lowRisk": "optional",
    "mediumRisk": "one-reviewer",
    "highRisk": "two-reviewers",
    "externalReviewEvidence": "proxy/advisory"
  },
  "reviewers": {
    "pi": {
      "enabled": true,
      "timeoutMinutes": 15,
      "tools": ["read", "grep", "find", "ls"]
    },
    "claude": {
      "enabled": true,
      "timeoutMinutes": 20,
      "permissionMode": "plan",
      "tools": ["Read", "Grep", "Glob"],
      "maxBudgetUsd": 3
    },
    "codex": {
      "enabled": false,
      "timeoutMinutes": 15,
      "note": "Same-family fallback only; not a replacement for independent review."
    }
  }
}
```

## Desired Commands

The existing review-pack commands remain the execution layer:

```bash
codex-orchestrator pack review --package-id PKG --task-id TASK --output review-pack/PKG
codex-orchestrator review run --package-id PKG --reviewer pi --pack review-pack/PKG
codex-orchestrator review import --package-id PKG --reviewer deepseek --file review.md --status passed
```

The policy layer should add:

```bash
codex-orchestrator review policy show --repo .
codex-orchestrator review policy check --repo . --risk medium --task-count 4 --json
```

`review policy check` should answer:

- whether review is required;
- why the package falls into low/medium/high risk;
- which reviewers should run;
- which reviewers are unavailable on this machine;
- whether manual import is needed;
- whether the result is local/static policy evidence only.

## Ledger Requirements

The ledger should eventually record package review state, not only task-level
routine runs:

```json
{
  "packageId": "CUSTOMER-COUPON-CHECKOUT",
  "lane": "Customer ordering / coupon checkout",
  "taskIds": ["TASK1", "TASK2", "TASK3"],
  "risk": "medium",
  "reviewRequired": true,
  "reviewStatus": "reviewed",
  "reviewers": [
    {
      "name": "pi",
      "status": "passed",
      "reportPath": "review-pack/CUSTOMER-COUPON-CHECKOUT/pi-review.md",
      "evidenceLabel": "proxy/advisory"
    }
  ]
}
```

Until package-level ledger records exist, external review runs should continue to
be stored as `RoutineRun` entries with `packageId`, `reviewer`, and `reportPath`.

## Orchestrator Behavior

After each worker closeout:

1. Re-read ledger and current product lane.
2. If the current lane reaches a package boundary, run `pack review`.
3. Run `review policy check` to decide whether zero, one, or two reviewers are
   needed.
4. Invoke available configured reviewers with `review run`.
5. If a reviewer is unavailable, mark the review setup as blocked and allow
   `review import`.
6. Do not mark the package complete/stable/pre-ready until required review state
   is recorded or explicitly waived.

## Non-Goals

- No package-manager distribution work.
- No background daemon in this step.
- No automatic merge authorization from model agreement.
- No direct runtime, device, provider, pre/prod, or payment proof.
- No default `claude ultrareview`.
