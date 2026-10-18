# ADR-70022: Two-Tier Work-Queue Clear with a Guarded Hard Reset

**Date**: 2026-10-18
**Status**: Draft (PR #70022 is not yet accepted)
**Deciders**: Not recorded; pending maintainer review of PR #70022.

---

### Context

The Git-backed work queue (`pkg/workqueue`) is an append-only ledger. Its entries record Work, Claims, delivery barriers, admission identities, fairness charges, and native reservations. Operators had `cancel-work` for individual items and `control` for pausing admission or grants. They had no single operation to empty a queue. Emptying a queue carries two separate risks. Cancelling unfinished Work must not destroy the audit trail or reconciliation evidence for effects that are still in flight. Discarding the ledger completely would forget task and request identities, so old submissions could be accepted again and settled effects could repeat. Concurrent admissions and lost publication acknowledgments also make retries risky: a retried clear must never sweep Work admitted after the original request.

### Decision

We will add an administrator-only `gh aw work-queue clear --reason CODE` command with two modes:

1. **Default (history-preserving) clear.** The command builds one atomic `cancel_work` request that terminally cancels every `available` or `claimed` Work in the snapshot. It reuses existing cancellation operations and fences each claimed item on its current Claim. Completed Work, unresolved delivery barriers, admission identities, provenance, fairness charges, and native reservations are retained. If the snapshot contains more Work than the installed policy's atomic operation limit allows, the command fails and cancels nothing.
2. **`--hard` reset.** `Branch.HardClear` (`pkg/workqueue/hard_clear.go`) replaces the active ledger with a fresh `Genesis` commit. It then adds control requests that pause admission and grants and carry over the credential generation, followed by deployment requests that keep currently unavailable workers unavailable. The reset is allowed only when admission and grants are already paused, every native reservation has been released, and no Claim is open, no delivery barrier is pending or failed, and no effect is uncertain. Publication uses compare-and-swap and is never retried against a newer snapshot. The prior ledger stays reachable as the Git parent, and its SHA is returned. The reset intentionally forgets task and request identities, observations, historical worker revisions, and fairness debt.

Both modes, including `--dry-run` previews, refuse to run in CI before any authentication, read, or write. A stable `--request-id` recovers only the original accepted operation. For a hard reset, the request ID and prior SHA are encoded in the new policy epoch, so recovery still works after checkpointing.

### Alternatives Considered

#### Alternative 1: Implement hard clear with existing ledger compaction/checkpointing

Compaction already rewrites the ledger into a smaller form, so it could have been extended to drop Work. This was rejected because compaction preserves semantics: it keeps identities and fairness state. Using it to discard state would blur the boundary between a safe maintenance operation and a destructive reset. A separate fresh-Genesis path makes the loss of identities explicit and keeps the old ledger intact in Git ancestry.

#### Alternative 2: Offer only the history-preserving cancel-all (no hard mode)

Only atomic cancellation would avoid any loss of identities or debt. However, cancelled Work, request identities, and fairness debt would remain in the ledger forever, so operators could never return a queue to a clean state, for example after test runs or a corrupted policy history. Adding a hard mode with strict preconditions (paused, fully settled) and explicit warnings covers that need without making destructive behavior the default.

#### Alternative 3: Batch cancellation across multiple requests for large queues

Splitting large snapshots across several requests would remove the 256-operation ceiling. It was rejected because cancellation would no longer be atomic: a failure partway through would leave the queue partially cleared, and later batches could race with new admissions. The command instead fails and directs operators to pause the queue and use `cancel-work` in bounded batches.

### Consequences

#### Positive
- Operators have one auditable command to drain unfinished Work without losing completed Work, provenance, or reconciliation evidence.
- The hard reset cannot discard evidence for in-flight or uncertain effects. Its preconditions require a paused, fully settled queue, and the prior ledger is still recoverable through Git ancestry.
- Idempotent request IDs combined with compare-and-swap publication prevent duplicate cancellation or reset sweeps, even after lost acknowledgments or checkpoints.
- Refusing to run in CI, before any I/O, keeps automated workflows from accidentally invoking a destructive command.

#### Negative
- After a hard reset, old task and request identities can be resubmitted, and settled effects can repeat. Operators must handle this risk themselves; the tool only warns about it.
- Default clear is capped by the policy's atomic operation limit (at most 256 Work), so large queues need manual, batched `cancel-work`.
- Recovering a hard clear depends on a specific encoding (the `hard_clear_<hash>_<sha>` policy epoch and fixed control-request indices). Future changes to Genesis or control layouts must preserve this encoding.
- Neither mode stops native runs. Operators must reconcile reservations separately before a hard reset can succeed.

#### Neutral
- Policy, current worker routes and availability, and the credential generation carry over into the fresh ledger; historical worker revisions do not.
- The implementation lives in new clear-specific files (`pkg/cli/work_command_clear.go`, `pkg/workqueue/hard_clear.go`). The shared command file changes only to register the new command.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
