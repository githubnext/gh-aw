# ADR-54846: Unify Outcome Classification Onto a Single OutcomeStatus Enum

**Date**: 2026-08-22
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/cli` maintained two parallel outcome classification types: `OutcomeResult` (a standalone `string` type with constants such as `OutcomeAccepted`, `OutcomeRejected`, etc.) and `OutcomeStatus` (embedded in `OutcomeEvaluation`). `OutcomeReport` carried both — a `Result OutcomeResult` field and the embedded `OutcomeEvaluation.OutcomeStatus` — so every evaluator had to set two fields that were semantically equivalent. This dual representation made safe-output evaluation ambiguous and caused JSONL output to emit duplicate classification data under both `result` and `outcome_status` keys, leaving downstream consumers uncertain about which field to trust.

### Decision

We will eliminate `OutcomeResult` and its constants entirely, extend the existing `OutcomeStatus` enum to cover the values previously unique to `OutcomeResult` (specifically `OutcomeStatusError`, which was `OutcomeError`), and remove the `Result` field from `OutcomeReport`. All evaluators will set `report.OutcomeStatus` (accessed through the embedded `OutcomeEvaluation`) as the sole classification field. JSONL output will emit only `outcome_status`, not `result`.

### Alternatives Considered

#### Alternative 1: Keep Both Enums, Add Synchronization Logic

Keep `OutcomeResult` and `OutcomeStatus` as separate types, but introduce a mapping function that sets both fields consistently whenever an evaluator sets one. This would prevent API breakage for consumers of `Result` but leaves the dual-representation ambiguity in place and adds an indirection layer that future maintainers must remember to invoke.

#### Alternative 2: Deprecate OutcomeStatus in Favor of OutcomeResult

Reverse the direction: remove `OutcomeStatus` from `OutcomeEvaluation` and standardize on `OutcomeResult`. This avoids the breakage direction chosen, but `OutcomeEvaluation` already carried normalized signal and evidence-strength metadata not present on `OutcomeResult`, so going this route would require re-introducing those fields under a different home — net complexity gain with no benefit.

### Consequences

#### Positive
- Single source of truth for outcome classification; evaluators set exactly one field and there is no ambiguity about which value is authoritative
- Cleaner serialized schema: `OutcomeReport` JSON exposes only `outcome_status`; JSONL audit entries no longer emit a duplicate `result` key alongside `outcome_status`
- Reduced cognitive overhead — readers of evaluator code no longer need to track two parallel classification fields and their relationship

#### Negative
- Breaking schema change for existing JSONL consumers that read the `result` field; any pipeline or dashboard filtering on `result` must migrate to `outcome_status`
- High diff volume across many evaluator files, even though each individual change is a mechanical rename with no semantic complexity; this increases review surface area

#### Neutral
- `OutcomeStatusSkipped` was already present in `OutcomeStatus`; this change absorbs it into the unified enum with no behavioral change, but its presence is now formally part of the consolidated domain invariant verified by tests

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
