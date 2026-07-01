# ADR-42735: Attach Experiment/Variant Provenance to Audit Report Records

**Date**: 2026-07-01
**Status**: Draft
**Deciders**: Unknown

---

### Context

The audit pipeline emits `ReportProvenance` structs embedded in every report record type (`MissingToolReport`, `NoopReport`, `MissingDataReport`, `MCPFailureReport`). Previously these structs carried only `timestamp`, `workflow_name`, and `run_id`. As A/B experiment infrastructure matured, downstream consumers—dashboards, correlators, bug-triage scripts—needed to slice audit failures by experiment variant without a separate join to the experiment `state.json` artifact. The provenance struct is the only per-record metadata surface, and it already flows through every extraction path, making it the natural place to attach experiment context at extraction time.

### Decision

We will extend `ReportProvenance` with two optional flat string fields—`experiment_name` and `variant`—and populate them at record-construction time by reading `state.json` from the run directory. When multiple experiments are active simultaneously, we will select the **alphabetically first** experiment name and its assigned variant as the canonical provenance pair. Extraction helpers that construct `ReportProvenance` are consolidated into a single `buildReportProvenance` helper to ensure consistent wiring across all record types.

### Alternatives Considered

#### Alternative 1: Embed the full experiment assignments map

Extend `ReportProvenance` with an `Assignments map[string]string` field that captures every active experiment. This is more complete but changes the flat JSON shape of every record type, requiring updates to all downstream consumers, dashboards, and MCP tool output schemas. It also introduces a variable-length field in a type designed for flat serialization, complicating analytics queries that expect a fixed column structure.

#### Alternative 2: Use last-run-record experiment assignment (temporal selection)

Select the experiment assignment from the most recent run record in `state.json` (matching by `run_id` or latest timestamp) rather than alphabetical order. This would be more contextually meaningful when run IDs are available, but the existing `extractExperimentData` API does not expose per-run-ID lookup at record-construction time without additional refactoring. Alphabetical selection delivers determinism at lower implementation cost and is consistent with the existing `firstExperimentAssignment` helper pattern already used in formatting code.

### Consequences

#### Positive
- Downstream audit analysis can group or filter report records by `experiment_name` / `variant` using flat JSON field access—no secondary join to `state.json` required.
- Provenance fields remain omitempty; records produced by non-experiment workflow runs are unaffected (zero-value fields are omitted from JSON output).
- `buildReportProvenance` eliminates four separate inline `ReportProvenance{}` struct literals, reducing copy-paste risk when new record types are added.
- Deterministic: alphabetical sort produces identical output across runs regardless of Go map iteration order.

#### Negative
- When more than one experiment is active for a run, only the alphabetically first experiment/variant pair is recorded; the remaining assignments are silently dropped from per-record provenance.
- The alphabetical selection heuristic may not align with the most analytically significant experiment if experiment names are not chosen with sorting in mind.

#### Neutral
- `extractMCPFailuresFromLogFile` gained an optional `runDirOverride` variadic parameter to supply the run directory, since log files may be in a subdirectory; this preserves backward compatibility with call sites that pass only the log path.
- The MCP tool description for `audit_run` was updated to document the new `experiment_name` and `variant` fields on `missing_tools` and `mcp_failures` record types.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
