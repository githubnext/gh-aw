# ADR-71495: Additive First-Use Pool Installation for AW-Managed Work Queues

**Date**: 2026-10-24
**Status**: Draft
**Deciders**: [TODO: verify] pelikhan, Copilot coding agent

---

### Context

The smoke dispatcher failed because the `smoke-work-queue` pool was configured in `aw.json` and compiled into the caller's Policy proposal, but was absent from the Policy already installed on the queue branch. Before this PR, an installed Policy could only change through an administrator-only `policy` request, which needs a quiescent queue. As a result, adding a new pool to an existing AW-managed queue blocked new work. Trusted controls also reported both the submission and the dispatch failure as unclassified errors. Any fix must keep the work-queue invariants in place: existing pools, accepted work, and the Policy epoch must not change, and worker routes must stay compiler-approved and repository-scoped (see ADR-64955 and ADR-67322).

### Decision

We will add a new `PoolInstall` queue operation, carried by a new `pool_install` request intent, that adds exactly one new pool to an existing AW-managed Policy. It does not touch installed pools, accepted work, or the Policy epoch. Trusted submit and dispatch paths (`work_queue_dispatch.cjs`) call `installWorkQueuePools` on first use of a missing pool. Before publishing, it checks four things: the compiled proposal matches the installed Policy except for the new pool, the pool exists in the proposal, every profile is compiler-approved (listed in `work_queue_workflows` and `aw_context_workflows`) and repository-scoped with no `principal`, and each worker route passes `verifyWorkerRoute`. Replay in both Go (`installReplayPool`) and JS repeats the additive-only and repository-scope checks. The `administrator`, `producer`, and `dispatcher` roles may emit the operation. A TLA+ model (`specs/work-queue/QueuePoolInstall.tla`) bounds the safety argument. When installation cannot happen, failures are classified as `work_queue_pool_not_installed`, `work_queue_worker_not_compiler_approved`, or `work_queue_policy_proposal_mismatch`.

### Alternatives Considered

#### Alternative 1: Diagnostics only, recovery requires an administrator Policy update

Report `work_queue_pool_not_installed` with recovery guidance, and require an administrator to quiesce the queue and publish a full `policy` update. This was the PR's original scope, as described in the PR body. It keeps Policy mutation administrator-only, but every new pool would need manual, quiescence-gated intervention, which conflicts with first-use bootstrap semantics. It was rejected in favor of a narrowly scoped automatic path. Economic changes and changes to existing pools still use this path.

#### Alternative 2: Let first use republish the full compiled Policy

Treat a mismatched Policy like an absent branch and install the full compiled proposal. This is simpler, but it could silently change existing pools, economics, or the Policy epoch while work is in flight, which breaks the immutability and quiescence guarantees. It was rejected.

#### Alternative 3: Separate queue per pool

Give each pool its own queue branch so that adding a pool always bootstraps a new queue. This avoids mutating installed Policy, but it splits scheduling and ledger state and departs from the single-Policy queue model. It was rejected.

### Consequences

#### Positive
- Adding a pool to `aw.json` and recompiling is enough to make it usable on an existing AW-managed queue. No administrator quiescence step is needed.
- Missing pools, unapproved worker routes, and proposal mismatches now produce classified, actionable errors instead of unclassified failures.
- Existing pools, accepted work, and the Policy epoch are preserved, enforced at publish time and again at replay, with a bounded TLA+ model.

#### Negative
- Non-administrator roles (`producer`, `dispatcher`) can now mutate installed Policy, but only additively. This widens the Policy-write surface and depends on the compiler-approval and route checks being correct.
- The protocol surface grows: a new operation kind, a new intent, new JSON schemas, and updated canonical fixtures. Go and JS replay must stay in sync.
- The PR description says live recovery requires an explicit administrator update and that the PR "does not mutate the live queue". The final code does append `PoolInstall` commits on first use. [TODO: verify the intended scope and reconcile the PR description.]

#### Neutral
- Changing an existing pool or economic Policy still requires an explicit administrator `policy` update on a quiescent queue.
- Pool installation is only for AW-managed (`authorization: "aw"`) queues. Other queues report `work_queue_pool_not_installed`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
