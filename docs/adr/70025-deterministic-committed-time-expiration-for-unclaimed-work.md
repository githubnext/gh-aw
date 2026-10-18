# ADR-70025: Deterministic Committed-Time Expiration for Unclaimed Work Queue Entries

**Date**: 2026-10-18
**Status**: Draft
**Deciders**: pelikhan, Unknown (TODO: verify reviewers)

---

### Context

Work queue entries that are never claimed currently stay eligible for scheduling forever, so stale Work accumulates and can be granted long after it stops being useful. The work queue is an event-sourced log: state is a projection rebuilt by replaying committed operations (`Replay`). It is also compacted into checkpoints and implemented twice, once in Go (`pkg/workqueue`) and once in JavaScript (`actions/setup/js/work_queue_*.cjs`), and the two must produce identical projections. Any expiration mechanism therefore has to give the same result on every replaying machine, survive checkpoint/suffix replay, and stay identical across both implementations. It also must not weaken the authority of Claims that are already running (native reservations, fairness charges, delivery fencing, Results).

### Decision

We will add an optional, immutable `expires_at` deadline (Unix milliseconds) to Work definitions. Expiration will be evaluated against a monotonic **committed-time watermark** (the maximum `At` of committed events), not the wall clock. When the watermark reaches the deadline (inclusive), Work that is still `available` and is not in `pending`/`failed` delivery moves deterministically to `cancelled` with reason `work_expired`. The watermark is stored in checkpoints so that replaying a checkpoint plus its suffix matches a full replay. Live readers (CLI `work` commands, scheduler/activation) get a throwaway current-time view through `ReplayAt` / `expirationView`, which must never be used to build checkpoints. Before publishing a Claim grant, the selected deadline is checked again against `max(now, watermark)`, so expired Work is not granted. Running Claims never expire.

### Alternatives Considered

#### Alternative 1: Wall-clock evaluation during replay

Each reader could compare `expires_at` with `time.Now()` while replaying. This is the simplest option, but then the projection depends on when and where it is replayed. That breaks determinism, checkpoint/suffix equivalence, and Go/JS parity checks, so we rejected it. The PR still uses the wall clock, but only for disposable live views and the pre-publication recheck.

#### Alternative 2: Explicit expiration operations (sweeper / garbage-collection commits)

A scheduled job or the scheduler could append explicit `Cancel` operations for Work past its deadline. This keeps replay purely event-driven and would not need a new watermark field. We rejected it because it needs a separate deadline service with its own liveness and authority, and because Work stays grantable until the sweeper runs. [TODO: verify whether this was evaluated by the author]

#### Alternative 3: Lease/timeout on running Claims as well

Deadlines could also apply to running Claims, releasing their native reservations on timeout. The PR explicitly rules this out because it would weaken Claim authority and could release capacity or fabricate outcomes without verified evidence. The TLA+ mutation `BrokenExpirationRelease`/`BrokenExpirationClaim` configurations encode this as a violation.

### Consequences

#### Positive
- Stale unclaimed Work stops being schedulable and stops blocking dependency refresh, without a separate cleanup service.
- Historical replay stays deterministic and machine-independent, and checkpoints keep the watermark, so checkpoint and suffix replay stay equivalent.
- Running Claims, fairness charges, native reservations, and Results are unaffected, which keeps existing safety guarantees.
- Bounded formal evidence (`specs/work-queue/WorkExpiration.tla`, 34,023 states at depth 6, mutation controls) backs the safety properties.

#### Negative
- Expiration only moves forward when new events are committed. Without new commits, the persisted projection does not record Work as expired, and readers depend on the current-time view. The PR states there is no hard UTC publication fence.
- Old readers must be upgraded before deadline-bearing Work is published, because the protocol uses closed-schema validation (new `expires_at` in 7 generated schemas).
- The logic is duplicated in Go and JavaScript (`expiration.go` / `work_queue_expiration.cjs`), which adds to the parity maintenance burden.
- Deadlines are immutable. Changing one means creating a new Work node identity (`work_conflict`).

#### Neutral
- There are now two projection flavours: committed-time `Replay` (authoritative, checkpointable) and `ReplayAt` (live, disposable). Callers must pick the right one.
- `expires_at` is exposed through the CLI (`--expires-at`), graph, staged-plan, and MCP submission paths, and documented in the work-queue reference and specification.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
