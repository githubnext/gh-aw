# ADR-26148: Deterministic Audit Metrics via run_summary.json Cache and workflow-logs/ Exclusion

**Date**: 2026-04-14
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `audit` command reported wildly inconsistent `token_usage` and `turns` on repeated invocations for the same workflow run (observed: 9 turns / 381k tokens on one call, 22 turns / 4.7M tokens on another). Two compounding bugs caused this: (1) `AuditWorkflowRun` unconditionally re-processed all local log files on every call, even when a fully-computed `run_summary.json` was already on disk; and (2) the log-file walk in `extractLogMetrics` did not exclude the `workflow-logs/` directory, which `downloadWorkflowRunLogs` populates with GitHub Actions step-output — files that capture the same agent stdout already present in the agent artifact logs, inflating token counts by approximately 12×.

### Decision

We will adopt a **cache-first strategy** for `AuditWorkflowRun`: before performing any API calls or log processing, check whether a valid `run_summary.json` exists on disk (validated by CLI version). If a cache hit is found, reconstruct `ProcessedRun` from the cached summary and return immediately via a shared `renderAuditReport` helper, bypassing all re-download and re-parse logic. We will additionally **exclude the `workflow-logs/` directory** from the `extractLogMetrics` log walk by returning `filepath.SkipDir` whenever the walk visits a directory named `workflow-logs`, preventing GitHub Actions runner captures from being counted as agent artifact data. Together, these two changes ensure that repeated `audit` calls for the same run produce identical metrics.

### Alternatives Considered

#### Alternative 1: Invalidate and Overwrite the Cache on Every Call

Rather than treating the cached `run_summary.json` as authoritative, re-process logs on every call and overwrite the cache. This would keep the cache "fresh" but would perpetuate the inconsistency problem: log re-processing can produce different values depending on which files are present at the time (e.g., if `workflow-logs/` has been populated between calls). This was rejected because consistency of audit metrics across repeated calls is the primary requirement.

#### Alternative 2: Exclude `workflow-logs/` Files by Name Pattern Instead of Directory Skip

Rather than skipping the entire `workflow-logs/` directory with `filepath.SkipDir`, selectively exclude individual files whose names match known GitHub Actions runner-log patterns (e.g., `*_Run log step.txt`). This would be fragile: GitHub Actions file naming conventions can change, and any unrecognized file would silently inflate metrics again. Skipping the entire directory by name is simpler, robust, and aligns with how `downloadWorkflowRunLogs` places its output.

#### Alternative 3: Store Canonical Metrics in a Separate Lock File

Record only the metrics (token usage, turns) in a dedicated lock file separate from `run_summary.json`, and read that lock file on subsequent calls. This adds file-system complexity without meaningful benefit over reusing the existing `run_summary.json` structure. The current `loadRunSummary` already performs CLI-version validation, providing a clean automatic invalidation mechanism.

### Consequences

#### Positive
- Repeated `audit` calls for the same run are now deterministic and produce identical output.
- The cache-hit path avoids all API calls and file re-parsing, making subsequent audits significantly faster.
- The `renderAuditReport` helper function eliminates the duplicated render + finalization logic that previously existed in both the fresh-download and (now) cache-hit code paths.
- Cache invalidation on CLI upgrade is automatic via the existing `CLIVersion` check in `loadRunSummary`.

#### Negative
- The first successful `audit` call becomes the canonical source of truth. If log files were incomplete on the first run (e.g., partial download), the cached metrics will be wrong until the cache is manually cleared or the CLI is upgraded.
- The `workflow-logs/` exclusion is a directory-name-based heuristic. If `downloadWorkflowRunLogs` ever changes the output directory name, the exclusion silently stops working.
- Adding a new top-level helper (`renderAuditReport`) increases the surface area of the package's internal API.

#### Neutral
- The `run_summary.json` format is unchanged; only the read/write ordering is adjusted (save-before-render in the fresh-download path).
- Existing tests for `loadRunSummary` and `saveRunSummary` remain valid; new regression tests were added for the cache-hit path and the `workflow-logs/` exclusion.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Cache-First Audit Strategy

1. Implementations **MUST** check for a valid `run_summary.json` on disk before initiating any API calls or log-file processing in `AuditWorkflowRun`.
2. Implementations **MUST** treat a cache hit (valid `run_summary.json` with matching `CLIVersion`) as the authoritative source of metrics and return immediately without re-processing logs.
3. Implementations **MUST NOT** overwrite an existing `run_summary.json` when serving a cache hit; the cached file **MUST** remain unmodified.
4. Implementations **MUST** persist `run_summary.json` to disk before calling the render step in the fresh-download path, so that a render failure does not prevent future cache hits.
5. Implementations **SHOULD** log a message (at the appropriate verbosity level) indicating that a cached summary is being used, including the original `ProcessedAt` timestamp.

### Log Metric Extraction

1. Implementations **MUST** skip the `workflow-logs/` directory (and its contents) when walking the run output directory in `extractLogMetrics`.
2. Implementations **MUST** use `filepath.SkipDir` (or equivalent) to exclude the entire `workflow-logs/` subtree, not individual files within it.
3. Implementations **MUST NOT** include token-usage data found in `workflow-logs/` in the `LogMetrics.TokenUsage` or `LogMetrics.Turns` totals.
4. Implementations **MAY** log a debug message when skipping the `workflow-logs/` directory to aid in future diagnostics.

### Shared Render Path

1. Implementations **MUST** use a single shared function (currently `renderAuditReport`) to build and emit the audit report, invoked by both the cache-hit path and the fresh-download path.
2. The shared render function **MUST NOT** re-extract metrics from log files; it **MUST** use only the metrics passed to it by the caller.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24396807146) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
