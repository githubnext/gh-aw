# ADR-70025: Deterministic Committed-Time Expiration for Unclaimed Work Queue Entries

**Date**: 2026-10-18
**Status**: Proposed (implemented in PR #70025; maintainer acceptance pending)
**Decision owner**: @pelikhan, the PR author. Formal architecture approval is not implied by this record.

---

### Context

Without a deadline, unclaimed Work can remain eligible indefinitely and be granted after it stops being useful. The requested scope is expiration of unclaimed Work, not leases for running Claims. The work queue is an event-sourced log: state is a projection rebuilt by replaying committed operations (`Replay`). It is compacted into checkpoints and implemented in both Go (`pkg/workqueue`) and JavaScript (`actions/setup/js/work_queue_*.cjs`). Expiration must produce identical historical projections across readers, clock changes, and checkpoint/suffix replay, without weakening Claim authority, native reservations, fairness charges, delivery fencing, or Results.

### Decision

Add an optional, immutable `expires_at` deadline to Work definitions: a nonnegative safe-integer Unix-millisecond timestamp; omission means no expiration. Historical replay advances a monotonic **committed-time watermark**, the maximum committed `QueueCommit.at`, before and after each commit. At `watermark >= expires_at`, Work that is `available` and outside `pending`/`failed` delivery becomes `cancelled` with reason `work_expired`. This is a derived transition, not a synthetic `WorkCancellation` record. The immutable deadline and committed time supply its authority without a redundant cancellation commit or external timer.

Checkpoints persist the maximum in `last_at`, not merely the final prefix timestamp, and preserve deadlines and derived cancellations. Restoration also takes the maximum from retained receipt timestamps and the checkpoint envelope time, including for older checkpoints. Backdated commits cannot lower the watermark or resurrect expired Work.

Live readers use disposable trusted-time views through `ReplayAt`, `replayTransactions(asOf)`, or `expirationView`; these views must never become checkpoint authority. Publication rechecks selected deadlines against `max(now, watermark)` even if HEAD has not changed and regenerates a candidate when a deadline has elapsed. This is a trusted decision-time check, not a hard UTC publication fence.

Running Claims remain authoritative after their Work deadlines. Claim cancellation can return Work to availability, where it expires at the current watermark. Pending or failed retained delivery continues to fence expiration; verified retained delivery can expose elapsed Work to expiration instead of another Claim. Expiration neither releases native reservations nor refunds charges, fabricates Results, satisfies dependencies, or removes FIFO history and stable request identity.

### Alternatives Considered

#### Alternative 1: Wall-clock evaluation during replay

Each reader could compare `expires_at` with its wall clock while replaying. That would make historical projections depend on when and where they are read, breaking deterministic replay and checkpoint/suffix equivalence. Wall-clock evaluation is therefore limited to disposable live views and the pre-publication recheck, rather than historical replay.

#### Alternative 2: Explicit expiration operations (sweeper / garbage-collection commits)

Explicit `WorkCancellation` records, produced by a sweeper or the scheduler, were considered during the investigation. They keep cancellation directly recorded in the event log, but make expiration depend on an additional successful publication and add redundant records for a consequence already determined by the deadline and committed time. Without a separate selection-time check, Work remains grantable until that cancellation is committed. The derived transition avoids this extra publication dependency while retaining explicit cancellation for other authorized reasons.

#### Alternative 3: Lease/timeout on running Claims as well

Lease-like expiration of running Claims was also considered and excluded from this scope. Elapsed time does not prove physical worker quiescence or authorize native-capacity release. Such a design would need a separate draining/fencing protocol and termination evidence; simply timing out Claims would weaken existing authority. The `BrokenExpirationRelease` and `BrokenExpirationClaim` model configurations check that timeout-only release and expiration of running Claims violate the selected safety contract.

### Consequences

#### Positive
- Stale unclaimed Work stops being schedulable without a separate cleanup service; cancellation does not falsely satisfy its dependents.
- Historical replay stays deterministic and machine-independent, and checkpoints keep the watermark, so checkpoint and suffix replay stay equivalent.
- Expiration preserves running Claim authority, fairness charges, native reservations, Results, FIFO history, and stable requests.
- The [expiration model](../../specs/work-queue/WorkExpiration.tla) supplies bounded safety evidence: 34,023 distinct states at depth 6, with five negative mutations and two safety-guarded witnesses. It is not an unbounded proof, runtime refinement, or liveness theorem.

#### Negative
- Historical expiration advances only with committed time. Without queue activity, no timer publishes automatically; live readers use disposable time views. Clock skew, uncommitted clock rollback, or a deadline crossed during Git publication prevent a hard UTC fence.
- Old readers must be upgraded before deadline-bearing Work is published because the protocol uses closed-schema validation.
- The logic is duplicated in Go and JavaScript (`expiration.go` / `work_queue_expiration.cjs`), which adds to the parity maintenance burden.
- Deadlines are immutable. Changing one means creating a new Work node identity (`work_conflict`).
- Finite-deadline Work can lose eligibility before service; conditional eventual-service guarantees do not promise a grant or completion before expiration.

#### Neutral
- There are two projection flavours: committed-time replay (authoritative, checkpointable) and explicit live-time views (disposable). Callers must keep them separate.
- `expires_at` is exposed through CLI (`--expires-at`), graph, staged-plan, and MCP submission paths. The [reference](../src/content/docs/reference/work-queue.md#work-expiration) and [specification](../src/content/docs/specs/work-queue-specification.md#unclaimed-work-expiration) define the shared Go/JavaScript contract.

---

Implementation: [PR #70025](https://github.com/github/gh-aw/pull/70025). Verification scope and model limits: [expiration model evidence](../../specs/work-queue/README.md#unclaimed-work-expiration-model).
