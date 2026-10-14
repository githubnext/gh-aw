# ADR-68827: Route Trusted Work-Queue Dispatch Through the Controller and Build the Control Queue Client From the Publisher Token

**Date**: 2025-10-14
**Status**: Draft
**Deciders**: Unknown [TODO: verify — PR author @pelikhan and work-queue maintainers]

---

### Context

The work queue has no supported way for an operator to request that a *dispatcher* workflow be started: the only available CLI paths either assign workers or touch immutable Claims, so an investigation (the ESLint factory review) had to improvise outside the execution boundary. Separately, the control step for trusted queue operations constructed its queue client without a credential that qualifies for authentication-scoped caching, so every control invocation re-verified the full checkpoint history. On a representative fixture (32 ancestors, 3 deployment updates, 15 Work items, 100 retained requests) that cost 4,946 GraphQL-publication API calls and ~8.2 s locally, dominated by repeated Git commit/tree/blob reads. A third defect was observed in the same area: same-epoch compacted-Policy recovery failed for accepted control requests that have no compacted parameters digest.

### Decision

We will add a first-class `gh aw work-queue --repo OWNER/REPO dispatch DISPATCHER` command that resolves a dispatcher route at the repository's verified default revision (`pkg/workqueue/dispatcher.go`, `pkg/cli/work_command_dispatch.go`) and routes the request through the trusted controller, validating the dispatcher source and its compiled `.lock.yml` and correlating the native run — instead of assigning workers or rebinding immutable Claims. We will also construct the control-step queue client from the protected publisher token before the first queue load (`actions/setup/js/work_queue_control_adapter.cjs` reading `GH_AW_WORK_QUEUE_TOKEN`, bound by the compiler via the existing shared publisher-token resolver in `pkg/workflow/safe_outputs_env.go`), so that checkpoint-history verification becomes cache-eligible within an invocation. Worker-launch credentials remain a separate credential path, and preview mode constructs no credential client.

### Alternatives Considered

#### Alternative 1: Reuse existing worker-assignment / Claim commands for dispatch

Operators could continue to drive dispatchers through the existing assignment or Claim-manipulation surface. Rejected because dispatch is not a worker assignment: it would require granting or rebinding immutable Claims, which weakens the execution boundary the queue protocol is built on, and it offers no source/revision validation or native-run correlation.

#### Alternative 2: Keep injecting an arbitrary queue client and add a dedicated history cache

Instead of deriving the client from the publisher token, the control step could keep its current injected/dynamic-auth client and add a separate memoization layer for ancestry reads. Rejected because cache eligibility in this codebase is deliberately tied to a registered static-auth credential scope; a parallel cache would duplicate that mechanism and risk reusing verification proofs across differently authorized clients.

#### Alternative 3: Reduce history depth (shallower checkpoint verification)

Verifying fewer ancestors would cut the same API cost. Rejected because checkpoint verification semantics, fresh authorization/ref probes, and compare-and-swap behaviour are safety-critical and are explicitly left unchanged by this PR.

### Consequences

#### Positive
- Dispatch becomes a supported, validated operation (name pattern, frontmatter, compiled lock, verified default revision) rather than an ad-hoc workaround.
- ~93.8% fewer API calls on the GraphQL publication fixture (4,946 → 305) and ~11.6x faster local fixture time (8,181 ms → 705 ms); ancestry commit reads drop 2,080 → 65.
- End-to-end coverage added for claiming, native-run binding, retries, and performance, including offline tests against a captured live ledger and real Git history.
- Same-epoch compacted-Policy recovery now works for accepted control requests without a compacted parameters digest, while retaining actor and request-fingerprint checks.

#### Negative
- A new credential binding (`GH_AW_WORK_QUEUE_TOKEN`) is threaded into compiled workflows, requiring 33 lock files to be regenerated and widening the surface that must be reviewed for token-scope correctness.
- The dispatcher CLI adds a new trusted entry point that must be kept in lockstep with queue protocol changes and validated on every protocol upgrade.
- Performance now depends on cache eligibility: clients without an explicit publisher token stay uncached, so misconfiguration silently regresses to the slow path.
- The test suite grows substantially (large `.test.cjs` and Go test files) and parts depend on captured fixtures/profiles that are intentionally not committed.

#### Neutral
- Cache eligibility rules, checkpoint verification, fresh authorization/ref probes, and compare-and-swap semantics are unchanged; proof reuse is scoped to one invocation of a registered, unchanged static-auth client.
- Preview mode remains credential-free, so dispatch previews cannot validate launch credentials.
- Reported timings come from fake/offline transports and do not reflect real GitHub latency.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
