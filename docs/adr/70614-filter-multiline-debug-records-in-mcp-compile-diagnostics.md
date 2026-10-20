# ADR-70614: Filter Multiline Debug Records in MCP Compile Diagnostics

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: pelikhan, Unknown (PR #70614 reviewers)

---

### Context

The MCP `compile` tool captures compiler output and returns the actionable warnings and errors to the calling agent. When `DEBUG` logging is active, `pkg/logger` writes records shaped like `namespace message +elapsed`. Those records were filtered one line at a time with `compileDebugLinePattern`. Some records span several lines because they embed YAML or blank lines before the elapsed suffix. The elapsed suffix can also use either micro sign: `µ` (U+00B5) or `μ` (U+03BC). As a result, the continuation lines of a debug record leaked into the diagnostics. With `dry_run: true` and `strict: true`, that noise polluted the results for real workflows such as `agent-job-health` (issue #69877).

### Decision

We will recognize complete logger records while preserving ambiguous text. A namespace-like line starts a provisional buffer limited to the existing 4 KiB diagnostic budget. Namespace syntax is shared with the single-line matcher and includes delimiter-free names such as `bootstrap`. Only a complete record ending in an elapsed suffix is discarded. Both micro-sign variants are accepted.

Unconfirmed candidates are replayed through normal diagnostic extraction at EOF, on buffer overflow, before a new record, or at an unindented error, warning, or ShellCheck finding boundary. A later elapsed suffix cannot confirm a candidate across such a diagnostic boundary. Indented YAML and blank lines can remain part of a confirmed multiline record. This keeps memory bounded without changing compiler logging or suppressing subsequent actionable failures when a record is incomplete.

### Alternatives Considered

#### Alternative 1: Disable or redirect debug logging in the MCP compile subprocess

Compile with `DEBUG` unset, or send logger output to a separate stream, so no debug records reach the diagnostics at all. This avoids heuristic parsing, but changes the existing captured-output design or removes explicitly requested debug evidence. The scoped fix retains the current subprocess streams.

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
- Logger-shaped plain text is inherently ambiguous. Unconfirmed or oversized records are retained rather than silently discarded, so truncated debug text can remain in the bounded diagnostic result.
- Provisional buffering can delay diagnostic extraction until confirmation, a diagnostic boundary, the byte limit, or EOF.
- Correctness now depends on `pkg/logger` keeping its `namespace message +elapsed` format. A format change requires updating this filter.

#### Neutral
- The `agent-job-health` workflow instructions were changed to match its existing CLI-proxy transport, and its lock metadata was regenerated. Permissions did not change.
- The model inventory in the regression test comes from fixtures, so the test avoids live artifact queries.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
