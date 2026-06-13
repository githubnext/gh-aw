# ADR-39101: Aggregate All Usage-Artifact JSONL Files for Forecast AIC

**Date**: 2026-06-13
**Status**: Draft
**Deciders**: Unknown (generated from PR #39101)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The cost-forecast pipeline computes per-run AI Credit (AIC) cost from a single token-usage file produced by the main agent. As workflows began spending AIC in threat-detection steps, that spend was recorded in separate usage records inside the compact `usage` artifact and was never read by the forecast loader, so forecast totals silently undercounted real cost. The forecast issue also exposed only a single monthly P50 figure, hiding the spread of the Monte Carlo projection from anyone trying to reason about worst-case monthly spend.

### Decision

We will compute forecast AIC by aggregating **every** `.jsonl` file under a run's `usage` artifact directory rather than reading only the main agent usage file. For each record we prefer an explicit credit value (`ai_credits`/`aic`) and otherwise recompute AIC from raw token counts via `computeModelInferenceAIC`. When no usage-directory files are present we fall back to the existing single-file path, preserving backward compatibility. We will also widen the forecast report from a single `Monthly (P50)` column to `Monthly (Low/P50/High/Stdev)` derived from the Monte Carlo distribution.

### Alternatives Considered

#### Alternative 1: Keep reading only the main agent usage file

The status quo. Rejected because it structurally cannot see detection spend, which lives in sibling records within the `usage` artifact — the very gap that motivated this change. No amount of per-run scaling fixes an input that omits a cost source.

#### Alternative 2: Pre-aggregate AIC upstream into one summed file

Have the artifact producer emit a single pre-summed usage file the forecast loader reads as-is. Rejected for this change because it pushes cost-summation and AIC-recomputation logic into artifact generation, couples the forecast format to the producer, and is a larger blast radius than reading the files that already exist. Reading the directory keeps the forecast loader as the single owner of AIC computation.

### Consequences

#### Positive
- Forecast totals now include threat-detection credits, eliminating the documented undercount.
- Both explicit-credit and token-only usage records are handled, so detection records missing `ai_credits` still contribute cost via recomputation.
- The widened report surfaces low/high/stdev, letting readers gauge projection spread, not just the median.

#### Negative
- The loader now walks the entire `usage` directory per run, adding filesystem I/O and a `filepath.Walk` traversal that scales with artifact file count.
- Per-record precedence logic (`ai_credits` → `aic` → recomputed) adds branching that must stay in sync with the artifact record shape; a renamed field would silently zero a cost source.
- The forecast issue table is wider, consuming more horizontal space in the rendered report.

#### Neutral
- Behavior is unchanged for runs without a `usage` directory; the single-file path remains the fallback.
- Sorting and totals stay centered on monthly P50, so report ranking is unaffected by the added columns.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Forecast AIC Aggregation

1. When a run directory contains a `usage` subdirectory with one or more `.jsonl` files, the AIC-only loader **MUST** compute total AIC from all such files rather than from the single token-usage file.
2. For each usage record, an implementation **MUST** prefer an explicit credit value (`ai_credits`/`aic`) when present and positive, and **MUST NOT** also recompute AIC from token counts for that same record.
3. When no explicit credit value is present, an implementation **SHOULD** recompute AIC from the record's token counts using the shared inference-cost function.
4. When no `usage` directory files are found, an implementation **MUST** fall back to the existing single-file token-usage path.
5. Records that are malformed, empty, or non-AIC **MUST** be skipped without aborting aggregation of the remaining records.

### Forecast Report Shape

1. The forecast table **MUST** present `Monthly (Low)`, `Monthly (P50)`, and `Monthly (High)` as the Monte Carlo P10, P50, and P90 of 30-day total AIC respectively.
2. The forecast table **MUST** present `Monthly (Stdev)` as the Monte Carlo standard deviation of the 30-day total-AIC distribution.
3. Sorting and totals **SHOULD** remain centered on the monthly P50 value.

### Conformance

An implementation is conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/27471799541) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
