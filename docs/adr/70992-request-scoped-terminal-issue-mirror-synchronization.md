# ADR-70992: Request-Scoped Terminal Issue Mirror Synchronization for the Work Queue

**Date**: 2026-10-22
**Status**: Proposed
**Deciders**: @pelikhan (PR author); repository maintainers through review of [#70992](https://github.com/github/gh-aw/pull/70992)

---

### Context

The work queue mirrors each Work item to an existing GitHub Issue ([ADR-67025](67025-mirror-work-queue-work-to-github-issues.md)). It projects status labels and a canonical summary comment onto that Issue from receipts held in the queue ledger ([ADR-67649](67649-centralize-work-queue-receipt-projections.md)). Mirrors could keep stale `Queued` or `Running` labels after the authoritative Work had become terminal, for two reasons. First, a transient failure while reading projection metadata in the conclusion step (for example, an `UND_ERR_SOCKET` "other side closed" error) dropped the projection silently and did not fail the step. Second, terminal transitions made outside the original agent run had no projection hook. These include maintenance-workflow transitions and operator commands such as cancellation, `clear` and reconciliation. Any fix had to keep several existing guarantees: the keep-open Issue policy, immutable Issue and summary bindings, run/Claim authority boundaries, fences on ambiguous writes, and V3 ledger readers. Native mutations cannot be retried safely, because an unacknowledged write may already have taken effect.

### Decision

Synchronization is scoped to **exact committed terminal requests**, never a global repair sweep. Original hooks retain their original admission/Claim authority. Maintenance may batch only requests authored by its matching authenticated native run and attempt. Automatic operator synchronization is bound to its exact publication; it does not grant a writer general recovery authority. Maintainers or admins can recover one terminal request explicitly with `gh aw work-queue sync-issues --request-id REQUEST_ID`.

Synchronization updates only existing, same-repository AW-managed Issues and canonical summary comments, preserving checked run/Claim history and verified outcome links. It never creates, rebinds or closes Issues or comments. It acquires sorted Work and Issue projection locks as git refs, replays fresh ledger state under those locks, and checks immutable native identities before writing. Checkpoint trace normalization preserves terminal-request identity after compaction. Missing receipts remain per-Work pending without blocking other eligible mirrors.

Projection metadata reads use a bounded three-attempt retry budget. Native writes are never blindly retried. Definitive no-write rejections release acknowledged coordination; a verified duplicate-label rejection permits discovery of the existing label, not replay of an unknown write. Unknown or partial outcomes retain fences. Cleanup of safely acknowledged locks has a bounded context independent of caller cancellation. Mutation pacing is a rate-limit control, not an eventual-consistency workaround.

Pending projection remains visible as failure after required reporting and authoritative maintenance compaction complete; unresolved native reconciliation retains its blocking behavior. Diagnostics expose safe classifications and request metadata, never raw external exception messages. No new ledger operations or schema fields are added.

### Alternatives Considered

#### Alternative 1: Global reconciliation sweep of dirty or stale mirrors

A periodic or on-demand job could scan all Work, compare it with Issue labels, and repair any differences. This would also fix existing stale mirrors without anyone running a command. It was rejected because the sweep would need authority across many runs and Claims. It would also make blind decisions about uncertain writes, which risks stealing locks or overwriting the results of ambiguous mutations. The PR rules out global repair sweeps and taking over another run's admission or Claim authority.

#### Alternative 2: Record projection status in the ledger (new operation or schema fields)

A "projection pending" or "projection done" fact could be stored in the ledger, so that later readers could find and resume pending syncs. This would make pending state durable and easy to query. It was rejected because it changes the ledger format and would break existing V3 readers. The PR keeps the ledger schema unchanged and derives targets from existing IssueLink and summary receipts and terminal barriers.

#### Alternative 3: Retry all projection calls, including mutations

Wrapping every GraphQL call in the same retry budget would be simpler and would hide more transient failures. It was rejected because retrying label or comment mutations that were not acknowledged can apply them twice, or race with fences on writes whose outcome is unknown. Only reads are idempotent enough to retry blindly.

### Consequences

#### Positive
- Terminal transitions from maintenance and operator commands can update existing Issue mirrors without acquiring another run's authority. Recoverable failures remain explicit rather than silently retaining stale labels.
- Transient read failures are recovered within a bounded budget. Remaining failures turn into an explicit step failure with a clear recovery command, so they are no longer silently dropped.
- Authority stays narrow. Explicit recovery selects one committed request; maintenance batches only its own exact committed requests. Both update existing mirrors in the same repository, and identities are checked again under locks before any write.
- V3 ledger compatibility is preserved, because no new operations or fields are added.

#### Negative
- Mirrors that are already stale are not fixed automatically. An operator must find the exact request ID and run `sync-issues --request-id`, which adds work for maintainers.
- Projection locks stored as git refs add API round-trips. Unknown writes or uncertain releases deliberately retain fences. This decision does not authorize deleting or stealing them; any recovery requires separate verified evidence and authorization.
- The conclusion step can now fail even when the queue commit itself succeeded. This may surface as a workflow failure that needs triage.
- The new sync and terminal-target logic is a large amount of security-sensitive code, written in both Go (`pkg/workqueue/branch_issue_sync.go`) and JavaScript (`work_queue_issue_transitions.cjs`), and the two must stay consistent.

#### Neutral
- All lock files were regenerated so that the new projection-failure hook and the existing failed-Claim diagnostic retention are both present.
- Terminal status labels are derived from Work state: `Cancelled`, `Needs attention` (unverified or failed delivery barrier), `Needs review` (verified pull request output), or `Done` (other verified Result).
- Each sync is capped at 256 mirrors, and label pagination at 16 pages.

---

This proposal records the implementation decision; maintainer review and merge determine acceptance. It does not authorize production repair, workflow dispatch, or removal of retained fences.
