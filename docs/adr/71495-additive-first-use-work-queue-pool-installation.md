# ADR-71495: Additive First-Use Pool Installation for AW-Managed Work Queues

**Date**: 2026-10-24
**Status**: Draft
**Deciders**: [TODO: verify] pelikhan, Copilot coding agent

---

### Context

The smoke dispatcher failed because the `smoke-work-queue` pool was configured in `aw.json` and compiled into the caller's Policy proposal, but was absent from the Policy already installed on the queue branch. Before this PR, an installed Policy could only change through an administrator-only `policy` request, which needs a quiescent queue. As a result, adding a new pool to an existing AW-managed queue blocked new work. Trusted controls also reported both the submission and the dispatch failure as unclassified errors. Any fix must keep the work-queue invariants in place: existing pools, accepted work, and the Policy epoch must not change, and worker routes must stay compiler-approved and repository-scoped (see ADR-64955 and ADR-67322).

### Decision

`PoolInstall` adds exactly one new pool to an existing AW-managed Policy without touching installed pools, accepted work, or the Policy epoch. Ordinary submit and dispatch intents do not publish `PoolInstall` implicitly; they reject a missing pool before deployment synchronization. An explicit `pool_install` request checks the compiled proposal's stable Policy and existing-pool boundaries, resolves new profiles against repository-wide installed worker authority, and verifies each selected immutable route with `verifyWorkerRoute`. A caller's newer source ref does not replace installed profiles during installation; verified Deployment synchronization handles newer refs separately. Caller-local target checks still govern actual submission and dispatch. Replay in Go (`installReplayPool`) and JS requires each new profile to match an already installed same-named profile or one of its registered Deployment revisions, including ref, logical contract, identity, scopes, and scheduling authority. The new pool's `allowed_repositories` must also be a subset of the repositories authorized by installed pools. Reserved JavaScript pool keys are rejected consistently. The `administrator`, `producer`, and `dispatcher` roles may emit an explicit `PoolInstall` operation. A TLA+ model (`specs/work-queue/QueuePoolInstall.tla`) bounds the safety argument. Missing pools, unapproved worker routes, and proposal mismatches are classified as `work_queue_pool_not_installed`, `work_queue_worker_not_compiler_approved`, and `work_queue_policy_proposal_mismatch`.

### Alternatives Considered

#### Alternative 1: Implicit first-use installation

Allow ordinary submit and dispatch intents to create missing pools before proceeding. This makes untrusted intent data a trigger for durable Policy mutation and may perform that mutation before deployment synchronization. It was rejected; missing pools fail closed, while `PoolInstall` remains an explicit queue operation.

#### Alternative 2: Let first use republish the full compiled Policy

Treat a mismatched Policy like an absent branch and install the full compiled proposal. This is simpler, but it could silently change existing pools, economics, or the Policy epoch while work is in flight, which breaks the immutability and quiescence guarantees. It was rejected.

#### Alternative 3: Separate queue per pool

Give each pool its own queue branch so that adding a pool always bootstraps a new queue. This avoids mutating installed Policy, but it splits scheduling and ledger state and departs from the single-Policy queue model. It was rejected.

### Consequences

#### Positive
- A missing pool is rejected before submission, dispatch, or deployment synchronization; the error is classified as `work_queue_pool_not_installed`.
- Missing pools, unapproved worker routes, and proposal mismatches now produce classified, actionable errors instead of unclassified failures.
- Existing pools, accepted work, and the Policy epoch are preserved, enforced at publish time and again at replay, with a bounded TLA+ model.

#### Negative
- Non-administrator roles (`producer`, `dispatcher`) can now add pools, but cannot introduce worker authority through `PoolInstall`. Replay enforces that boundary even when a publisher bypasses the adapter.
- The protocol surface grows: a new operation kind, a new intent, new JSON schemas, and updated canonical fixtures. Go and JS replay must stay in sync.
- Explicit `PoolInstall` remains available to trusted `administrator`, `producer`, and `dispatcher` roles; it is not triggered by ordinary worker-facing intents.

#### Neutral
- Changing an existing pool or economic Policy still requires an explicit administrator `policy` update on a quiescent queue.
- Pool installation is only for AW-managed (`authorization: "aw"`) queues. Other queues report `work_queue_pool_not_installed`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
