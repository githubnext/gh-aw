---
description: Agent instructions for Git-backed work queue producers, dispatchers, workers and observers.
---

# Work Queue

`tools.work-queue` schedules immutable DAGs and Claim-scoped effects;
`work-queue.jsonl` is authoritative. V3 has no storage selector; Issues/PRs are
dependencies.
For issue-backed WorkQueueOps, see [WorkQueueOps](../../docs/src/content/docs/patterns/workqueue-ops.md).

## Select the role

- **Observer:** read/explain only; never submit, dispatch or finish Work.
- **Producer/dispatcher:** submit trusted plans and request bounded pool work;
  queue authority grants no arbitrary resource-writing outputs.
- **Worker:** require compiler-supplied version-3 `claims`; never override
  reserved `work_queue_assignment` or caller context.

Trust runtime roles, not snapshots. AW authenticates the native run, workflow and
revision; reruns cannot inherit attempt 1. Absent branches are uninitialized/empty;
reject malformed or unsupported logs.

## Configure deployment

Declare a worker with:

```yaml
tools:
  work-queue:
    require-assignment: true
    worker: true
```

Dispatchers need `tools.work-queue: true`, a worker allowlist and bounded
`dispatch-workflow.max`. AW supplies approved routes and credentials, binding
each Claim to its immutable native run; moved `target-ref` is ignored.

Optional `aw.json` `work_queue` configures scheduling and Issue settings
(concurrency, pending limit, retries, pools, fairness, label and title prefix).
Defaults: weighted-priority, one pool, concurrency 16, pending 4,096, singleton
assignments, three attempts, 30-second backoff. First submit installs compiled
Policy and Work. Standalone seeding is unsupported; AW and Actions authorize
trusted submissions. Economic Policy updates require quiescence and explicit
`policy --from-config --epoch EPOCH`; config edits never rewrite active state.
Compatible deployments evolve future execution without draining.
Never expose queue-write credentials to
agents/snapshot MCP. Repository access rules are GitHub's responsibility, not
queue configuration. See [deployment](../../docs/src/content/docs/guides/deploy-work-queue.md).

## Mirror admitted Work with Issues

`aw.json` `work_queue.issues: true` mirrors admitted Work with purple `work` and
`work:<status>` labels. Optional `label` customizes the tracking and status-label
prefixes; `title_prefix` defaults to `[work queue]`. Titles and summaries show
12-character Work IDs; Issue bodies show full IDs. Per-workflow
overrides/enrollment aren't required. Protected AW hooks project admissions/original
Claims only. Immutable `backing_issue` binds one Work per Issue.
Agents never write mirrors; human edits never establish Result. Pre-existing
Issues require exact installed `backing_issues` grants. Only trusted
admission-time `completion_policy` permits closure, never payload. Keep uncertain
writes pending; never recreate or globally repair. Upgrade closed-schema readers
first. See
[backing Issue reference](../../docs/src/content/docs/reference/work-queue.md#backing-issues).
Mirroring generates `.github/workflows/agentic-workqueue.yml` for Issue lifecycle
events outside agentic workflows; disabling it removes the workflow.

## Plan and dispatch

- Read the immutable snapshot; predictions may be stale and cannot override
  Policy. Submit bounded, secret-free plans within scheduling limits.
- Request a pool prefix with `work_queue_dispatch_next` (`pool`, `max_claims`,
  `max_dispatches`). Never choose winners/targets or use scalar dispatch.
- Trusted processing refreshes the ledger and commits a compatible fair prefix.
  CAS losers discard tentative choices/charges and recompute.

ESLint, daily-report and conformance agents only explain staged plans.
Trusted controls validate batches; empty dispatcher intents fail.
Unassigned dispatchers cannot emit `noop` or `report_incomplete`.

## Execute original Claims

- Process only the assignment's original Claims and stored plans. Scope every
  safe output and finish intent to one original Claim.
- Omit `claim_handle` only for an originally single-Claim assignment.
  Multi-Claim assignments require it on every message even after other members
  close. Reject malformed, null, foreign or conflicting selectors.
- Finish each member with `work_queue_claim_finish`, using
  `outcome: "completed"` or `"cancelled"`. Missing finish authorizes no effects;
  valid siblings settle independently.
- Default to one Claim; batch only compatible Work with reusable setup. A native
  batch conclusion is not every Work's Result.
- Treat finish as an intent: trusted processing publishes Completion before
  scoped effects and Result only after independently verified delivery.
- For a worker's effect beyond the built-in safe outputs, declare
  `safe-outputs.claim-adapters` (see
  [safe-outputs-runtime.md](safe-outputs-runtime.md)); the adapter binds one
  effect-type to an independently verified result, never to job success alone.

`<mcp-clis>` takes one JSON argument. `uninitialized` plus null SHA is empty;
first `work_queue_submit` installs Policy and Work. Submit before dispatch.
Policyless ledgers fail; workers require Policy; `status: "staged"` is no grant.

## Dependencies and recovery

Require fresh completed-state evidence for Issue predecessors and actual merge
evidence for PR predecessors; predecessors consume no Claims. Successors wait for
predecessor Results.

Never infer nonlaunch/termination from lost responses, dispatcher cancellation
or elapsed deadlines. Retain reservations until exact evidence exists. Do not
rerun effects after Completion with uncertain delivery; bounded verification
yields Result or DeliveryFailure. Compatible deployments preserve obligations,
debt, Results and frozen assignments; explicit `execution_ref` pins stay pinned.
Unavailable/incompatible workers pause affected Work only. Quiesce economic
Policy changes. Reject unsupported protocols and scalar assignments.
Retries/checkpoints preserve submission identity; never delete a queue to retry
admission.

## Load details only when needed

- Operator commands, TUI and diagnostic artifacts:
  [queue reference](../../docs/src/content/docs/reference/work-queue.md).
  Cancellation terminates Work, not workers; reconcile separately.
  Build `go build -o ./gh-aw ./cmd/gh-aw`; use
  `./gh-aw work-queue --repo OWNER/REPO stats --json`. `dispatch-next` grants
  reservations, never launches; use the authorized dispatcher.
- Daily report rotation and Claims:
  [portfolio walkthrough](../../docs/src/content/docs/patterns/daily-report-portfolio.md)
  and [shared worker instructions](../workflows/shared/daily-report-worker.md).
- Contracts and implementation boundaries:
  [specification](../../docs/src/content/docs/specs/work-queue-specification.md#91-implementation-coverage-and-remaining-requirements).
- Executable models, fixtures and bounded verification:
  [formal reference](../../specs/work-queue/README.md).

Tests do not prove Result or live deployment.
