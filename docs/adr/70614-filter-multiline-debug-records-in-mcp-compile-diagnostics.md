# ADR-70614: Filter Multiline Debug Records in MCP Compile Diagnostics

**Date**: 2026-05-14
**Status**: Draft
**Deciders**: pelikhan, Unknown (PR #70614 reviewers)

---

### Context

The MCP `compile` tool captures compiler output and returns the actionable warnings and errors to the calling agent. When `DEBUG` logging is active, `pkg/logger` writes records shaped like `namespace message +elapsed`. Those records were filtered one line at a time with `compileDebugLinePattern`. Some records span several lines because they embed YAML or blank lines before the elapsed suffix. The elapsed suffix can also use either micro sign: `µ` (U+00B5) or `μ` (U+03BC). As a result, the continuation lines of a debug record leaked into the diagnostics. With `dry_run: true` and `strict: true`, that noise polluted the results for real workflows such as `agent-job-health` (issue #69877).

### Decision

We will treat a logger debug record as a span that can cover several lines. The `mcpCompileDiagnostics` writer will track an `inDebug` state. A record starts on a line that either matches the full single-line pattern or starts with a namespaced token (`compileDebugStartPattern`, which needs at least one `:./-` separator). The record ends on the first line that ends with the elapsed suffix (`compileDebugEndPattern`). Every line in that span is omitted. Both micro-sign variants are accepted. We are keeping heuristic, stream-based filtering on the output, not changing how the compiler emits logs. This fixes the leak with a small, local change, and the writer still streams output without buffering it.

### Alternatives Considered

#### Alternative 1: Disable or redirect debug logging in the MCP compile subprocess

Compile with `DEBUG` unset, or send logger output to a separate stream, so no debug records reach the diagnostics at all. This is cleaner because no heuristic is needed. It was not chosen because the PR keeps the existing captured-output design. Splitting the streams would mean changing the logger or the process plumbing, and debug output could no longer coexist with diagnostics when a user explicitly enables it. [TODO: verify whether this was considered]

#### Alternative 2: Keep line-by-line filtering and extend the single-line regex

Widen `compileDebugLinePattern` to cover more cases. This would not work: a continuation line (YAML, a blank line) has no namespace prefix and no elapsed suffix, so a stateless per-line regex cannot recognise it as part of a debug record.

#### Alternative 3: Emit debug records in a structured format (e.g. JSON lines)

Make the logger emit records that can be parsed unambiguously. This would be robust, but it changes `pkg/logger` output for every consumer. That is out of proportion to a diagnostics-cleanup fix.

### Consequences

#### Positive
- Multiline debug records, including YAML continuation lines and blank lines, no longer leak into the MCP compile results.
- Both micro-sign encodings of the elapsed suffix are handled, so filtering no longer depends on the platform or encoding.
- A regression test compiles the real `agent-job-health` workflow twice with `dry_run` and `strict`. It checks deterministic locks and the batch gate.

#### Negative
- The filter is a heuristic state machine. A real compiler warning that starts with a namespaced token (for example `a:b ...`) would be treated as the start of a debug record. The filter would then hide it and keep hiding later lines until it sees a line ending in an elapsed suffix.
- If a record never ends with the elapsed suffix (truncated output), `inDebug` stays set and later actionable diagnostics could be hidden.
- Correctness now depends on `pkg/logger` keeping its `namespace message +elapsed` format. A format change requires updating this filter.

#### Neutral
- The `agent-job-health` workflow instructions were changed to match its existing CLI-proxy transport, and its lock metadata was regenerated. Permissions did not change.
- The model inventory in the regression test comes from fixtures, so the test avoids live artifact queries.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
