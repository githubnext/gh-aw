# ADR-70027: Metadata-Only Structured Diagnostics for Work-Queue Read Commands

**Date**: 2026-10-18
**Status**: Draft
**Deciders**: Unknown — pending maintainer review of PR #70027

---

### Context

Agents investigating `gh aw work-queue` problems relied on `replay --json` for detail, which returns the full reconstructed state including Work payloads and request history, or on `stats`, which only returned a handful of root counters (`work`, `available`, `claimed`, `dispatches`, ...). Neither answered common diagnostic questions cheaply: why a Claim was cancelled or delivery failed, whether a reservation is still retained versus merely historical, whether a pool can schedule its oldest available Work, and how active the ledger has been recently. `inspect` and `state` returned display text plus a few fields, so agents had to parse human-readable strings. The diagnostics must stay read-only (no reconciliation, no live Actions lookups, no effect replay), must not leak payloads, Result descriptors, request parameters or receipt bodies, and must keep existing JSON consumers working.

### Decision

We will add a dedicated diagnostics layer in `pkg/cli` (`work_command_diagnostics.go`, `work_command_inspection.go`) that derives typed, metadata-only reports from the replayed `workqueue.Projection` at a caller-supplied evaluation time. `stats --json` becomes a structured report (snapshot identity, Claim outcomes and cancellation/failure reasons, delivery states, retained reservations separated from historical dispatches, per-pool eligibility with a compact one-Claim/one-dispatch prediction, trusted admission ages, and a 24-hour committed-activity window) while preserving the original root counters; `inspect --json` and `state --json` gain typed `work`, `scheduling`, `claim`, and `dispatch` metadata alongside their existing `selection`/`details`/`text` fields; `explain --json` adds the same compact `prediction` shape. Absent evidence is reported as `null` rather than inferred, and the reports are computed purely from ledger snapshots.

Workflow `audit` and queue-enabled `logs` will reuse these diagnostics for validated version-3 activation artifacts. Summary ages, activity and scheduling predictions use the artifact's capture time, with `captured_snapshot` provenance, rather than querying the current queue. Original-assignment Work and Claim metadata preserves attempt, retry and delivery evidence without actor principals or payloads. Finish intents and later operation logs remain separate: neither advances activation-time counters or proves final delivery. The audit cache schema advances so older reports are regenerated; run-summary backfills compare diagnostic values rather than pointer identities.

### Alternatives Considered

#### Alternative 1: Direct agents to `replay --json` and let them post-process

The full replay already contains all underlying state, so no new code would be needed. It was not chosen because the output is large, includes Work payloads and request history that diagnostics should not surface, and pushes the derivation of eligibility, reservation retention, and admission age onto each consumer, producing inconsistent interpretations.

#### Alternative 2: Add a new diagnostics subcommand with its own schema

A separate command (e.g., `work-queue diagnose`) would avoid touching the existing `stats`/`inspect`/`state` outputs. It was not chosen because the existing commands are the established entry points, and additive fields on them preserve backward compatibility while avoiding another command surface to document and maintain. [TODO: verify whether a separate command was explicitly considered]

#### Alternative 3: Fetch live Actions status to enrich diagnostics

Querying live run status would let reports state whether reservations are actually running. It was rejected because the commands are defined as snapshot inspections; live lookups would make them non-deterministic, require extra permissions/network, and blur the line between ledger evidence and external state.

### Consequences

#### Positive
- Agents get compact (~5 KB), typed answers to common queue questions without parsing display strings or downloading payload-heavy replays.
- Payload privacy is enforced by construction and covered by regression tests (`assertWorkDiagnosticsPrivate`).
- Existing JSON fields, display text, pagination, and full replay remain unchanged, so current consumers are unaffected.
- Deterministic, time-parameterised functions (`workQueueStatsAt`, `workQueueInspectAt`) are straightforward to unit test.

#### Negative
- The `stats --json` output contract grows substantially; every new field becomes a de-facto public interface that is costly to change later.
- Diagnostic logic (eligibility, compact prediction, reservation retention) partly re-derives scheduler semantics in the CLI layer, which can drift from `pkg/workqueue` if not kept in sync.
- Snapshot-only reports can mislead if readers treat eligibility or `bound` dispatch state as proof of live execution; this must be stated in documentation.

#### Neutral
- Retired pools stay visible with `policy_installed: false`, zeroed limits, and prediction reason `pool_not_installed` rather than being hidden.
- `workRead` now falls back to the `--repo` flag when the projection has no repository recorded, which affects snapshot identity fields.
- Admission age uses only trusted admission timestamps; client-supplied enqueue timestamps are ignored, and missing or future timestamps are counted as unknown.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
