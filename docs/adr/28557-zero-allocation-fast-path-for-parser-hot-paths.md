# ADR-28557: Zero-Allocation Fast-Path Strategy for Parser Hot Paths

**Date**: 2026-04-26
**Status**: Draft
**Deciders**: pelikhan (copilot-swe-agent)

---

## Part 1 — Narrative (Human-Friendly)

### Context

Benchmark profiling revealed a 12–13.5% performance regression in `BenchmarkYAMLGeneration` and `BenchmarkParseWorkflow`. The root causes were: (a) `bufio.Scanner` being allocated on every parse invocation — 6+ instances per call, accounting for approximately 20% of total allocations — and (b) a redundant `os.ReadFile` call in `generateYAML` that re-read a file whose content had already been parsed and held in memory. Both issues occur in the hot path executed on every workflow compilation. The `pkg/parser` package is called on each compile, making allocation pressure and redundant I/O directly proportional to throughput.

### Decision

We will replace `bufio.Scanner` in parser hot paths with Go 1.24 zero-allocation string iterators (`strings.Lines` and `strings.SplitSeq`), add fast-path pre-checks (`hasIncludeDirectives`) that return early before any scanner or buffer is allocated when no directives are present, and cache the raw markdown body in `WorkflowData.RawMarkdown` so that `generateYAML` can compute the frontmatter hash from pre-parsed data rather than re-reading the file from disk. The fast-path pre-check strategy is the primary driver: most workflow files contain no `@include`/`@import`/`{{#import` directives, so the common case can be handled without any scanner allocation.

### Alternatives Considered

#### Alternative 1: `sync.Pool` for `bufio.Scanner` reuse

A pool of pre-allocated scanners could amortize the per-call allocation cost. This was considered because it requires no Go version change and keeps the existing scanning logic intact. It was rejected because it adds pool lifecycle complexity (reset state between uses, handle concurrent access), whereas Go 1.24 iterators are simpler, have zero ongoing cost, and the fast-path pre-check eliminates the allocation entirely for the common case rather than merely amortizing it.

#### Alternative 2: Pre-allocated fixed-size byte buffer

Allocating a reusable `[]byte` buffer at startup and passing it to a custom scanner would reduce heap pressure. This was not chosen because it requires manual buffer sizing, risks stack-overflow or truncation for unexpectedly large content, and still allocates the scanner struct itself; the `strings.Lines` iterator avoids all of this by operating directly on the string without a separate buffer.

#### Alternative 3: Optimize only the redundant file read (partial fix)

Addressing only the `generateYAML` re-read without changing the scanner strategy would recover 3–6% of the regression. This was rejected as a standalone approach because profiling showed the scanner allocations were the dominant cost (~20% of total allocations); a partial fix would leave the larger problem unaddressed.

### Consequences

#### Positive
- `BenchmarkParseWorkflow`: 11% throughput improvement, 39% memory reduction per operation; `bufio` allocations eliminated.
- `BenchmarkYAMLGeneration`: 3% throughput improvement, 6% memory reduction per operation; redundant disk read eliminated.
- The fast-path pre-check (`hasIncludeDirectives`) is a string-contains scan — O(n) but cache-friendly — and pays for itself whenever directives are absent (the common case).
- `RawMarkdown` caching in `WorkflowData` makes the data flow more explicit: content read once during parsing flows through to YAML generation without additional I/O.

#### Negative
- `strings.Lines` and `strings.SplitSeq` require Go 1.24, setting a hard minimum runtime version for this package.
- `generateYAML` now has two code paths (fast path using `RawMarkdown`, fallback using disk read), adding branching complexity that must be kept in sync when the hashing logic changes.
- The `hasIncludeDirectives` pre-check can produce false positives (content containing `@include` in a comment but not as a directive), causing unnecessary scanner allocation in those edge cases; however, the check is conservative and correct — it never produces false negatives.

#### Neutral
- The `RawMarkdown` field is added to `WorkflowData`, a central struct; callers that construct `WorkflowData` externally without setting `RawMarkdown` automatically fall back to the disk-read path via the explicit `else` branch, preserving backward compatibility.
- H1 header scanning in `ExtractWorkflowNameFromMarkdownBody` is now bounded to the first 64 lines (previously unbounded), which is a behavior change for pathologically large files but is semantically correct because H1 headers appear at the top of Markdown documents.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Fast-Path Pre-Checks

1. Functions that iterate over content lines **MUST** call `hasIncludeDirectives(content)` before allocating any `bufio.Scanner` or buffer when the only reason to iterate is to process include/import directives.
2. `hasIncludeDirectives` **MUST** return `true` if and only if the content string contains at least one of the substrings `@include`, `@import`, or `{{#import`.
3. Functions that return early via the fast path **MUST** preserve the behavioral contract of the full code path, including trailing-newline normalization for content mode and returning `"{}"` for tool-extraction mode.
4. Fast-path pre-checks **MUST NOT** be added to functions whose iteration purpose is not exclusively include/import directive processing.

### String Iteration in Hot Paths

1. New line-scanning code in `pkg/parser` **MUST** use `strings.Lines` or `strings.SplitSeq` (Go 1.24) instead of `bufio.NewScanner(strings.NewReader(...))` when the input is an in-memory string.
2. Implementations **MUST NOT** wrap an in-memory string in `strings.NewReader` solely to pass it to `bufio.NewScanner`; use the zero-allocation iterators instead.
3. When an upper bound on lines to scan is required (e.g., searching for an H1 header), the iteration **MUST** break after the configured maximum (currently 64 lines for H1 header extraction) to bound worst-case cost.

### `WorkflowData.RawMarkdown` Caching

1. Code that constructs `WorkflowData` from a parsed result **MUST** populate `RawMarkdown` with the raw markdown body (before include expansion) when that content is available in memory at construction time.
2. `generateYAML` **MUST** use `ComputeFrontmatterHashFromParsedContent` when `WorkflowData.RawMarkdown` is non-empty, and **MUST** fall back to `ComputeFrontmatterHashFromFileWithParsedFrontmatter` (disk read) when `RawMarkdown` is empty.
3. The fallback path **SHOULD** log a debug message identifying the file path when it reads from disk, to aid future performance investigations.
4. Callers that construct `WorkflowData` externally **MAY** leave `RawMarkdown` empty; the fallback path **MUST** remain functional in this case.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24954772505) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
