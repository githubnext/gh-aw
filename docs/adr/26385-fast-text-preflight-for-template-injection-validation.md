# ADR-26385: Fast Text Pre-flight Check for Template Injection Validation

**Date**: 2026-04-15
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `generateAndValidateYAML` function in the workflow compiler runs a template injection check on every compiled workflow. Before this change, the check was triggered by `unsafeContextRegex.MatchString(yamlContent)`, which returned `true` if any unsafe context expression (e.g., `${{ github.event.* }}`, `${{ steps.*.outputs.* }}`) appeared *anywhere* in the compiled YAML — including safe locations such as `env:` blocks. Complex workflows commonly use expressions like `${{ github.event.pull_request.number }}` in `env:` bindings as the compiler's normal safe output pattern, meaning the expensive `yaml.Unmarshal` of the full ~73 KB lock file was triggered even when no template injection risk existed.

### Decision

We will replace the broad `unsafeContextRegex.MatchString(yamlContent)` trigger with a new `hasUnsafeExpressionInRunContent(yamlContent)` function that performs a fast, line-by-line text scan to detect unsafe expressions only when they appear inside `run:` block content. The scanner is deliberately conservative: when the context is ambiguous (e.g., a bare `run:` key with no value), it returns `true` to preserve the existing safety guarantee. This approach reduces the cost of the common case where expressions exist only in `env:` blocks while keeping the full YAML parse reachable for any true violation candidate.

### Alternatives Considered

#### Alternative 1: Keep the broad regex trigger as-is

The existing `unsafeContextRegex.MatchString(yamlContent)` approach was simple and correct but too coarse: it fired on every workflow containing *any* unsafe expression, regardless of whether the expression was in a safe location. Benchmarks showed ~4.8 ms/op and 24,595 allocations for complex workflows. This was rejected because the performance cost was measurable and the false-positive rate was high for typical compiler output.

#### Alternative 2: Parse the full YAML unconditionally (remove the fast-path entirely)

Removing the fast-path entirely would simplify the code but would make template injection validation unconditionally expensive. Given that many workflows never have unsafe expressions in `run:` blocks, this would significantly increase compilation latency for the common case. It was rejected on performance grounds.

#### Alternative 3: AST-based YAML scan before full unmarshal

A partial YAML parser (e.g., extracting only `run:` node values with a streaming parser) could be more precise than a line-by-line text scan. This was considered but rejected because it would require either a bespoke streaming parser or a dependency on an additional library, adding complexity that the benchmark data does not justify. The text-based scanner with conservative fallback achieves the same safety guarantee at much lower implementation cost.

### Consequences

#### Positive
- Benchmark `BenchmarkCompileComplexWorkflow` improved from ~4.8 ms/op / 24,595 allocs to ~2.5 ms/op / 12,610 allocs (~49% reduction).
- Benchmark `BenchmarkSimpleWorkflow` improved from ~3.0 ms/op to ~1.9 ms/op (~37% reduction).
- The safety invariant is preserved: the scanner never produces a false negative because it falls back conservatively whenever context is ambiguous.

#### Negative
- The line-by-line text scanner is a custom parser and can drift from YAML semantics over time. Unusual YAML constructs (e.g., deeply nested block scalars, multi-document streams) may cause the scanner to miscategorise lines, though the conservative fallback limits the blast radius to false positives (unnecessary full parses), not false negatives (missed violations).
- The scanner adds ~80 lines of custom parsing logic that must be maintained alongside any future changes to YAML block scalar handling.
- Indentation-based state tracking in the scanner assumes standard YAML indentation conventions; tab-indented YAML would not be handled correctly (Go's YAML library normalises tabs, so this is low risk in practice).

#### Neutral
- The test suite grows by 13 additional cases covering safe `env:`-only patterns, unsafe inline/block/folded `run:` patterns, multi-step YAML, and chomping indicators.
- The `generateAndValidateYAML` call site now calls `hasUnsafeExpressionInRunContent` instead of the raw regex; callers that relied on the internal regex match behaviour must update their usage.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Template Injection Pre-flight Check

1. The pre-flight check **MUST** return `true` (triggering the full YAML parse) whenever an unsafe context expression appears in the content of a `run:` block.
2. The pre-flight check **MUST** return `false` only when the scanner has positively confirmed that no unsafe expression appears in any `run:` block content.
3. The pre-flight check **MUST NOT** return `false` based on ambiguous state — when the scanner cannot determine whether a line belongs to `run:` block content (e.g., a bare `run:` key with no value), it **MUST** treat the ambiguous case as `true`.
4. The pre-flight check **MUST** apply an initial fast-path: if `unsafeContextRegex.MatchString(yamlContent)` returns `false`, the pre-flight check **MUST** immediately return `false` without scanning line-by-line.
5. The scanner **MUST** handle `run: |`, `run: >`, `run: |-`, `run: |+`, `run: >-`, and `run: >+` block scalar indicators as the start of multi-line `run:` block content.
6. The scanner **MUST** handle inline `run:` values (where the shell command follows `run:` on the same line) by checking the inline value directly with the unsafe expression regex.
7. The scanner **MUST** handle `- run:` sequence item syntax in addition to plain `run:` map key syntax.

### Integration with the Compiler

1. The `generateAndValidateYAML` function **MUST** use `hasUnsafeExpressionInRunContent` (not the raw `unsafeContextRegex`) as the condition for setting `needsTemplateCheck`.
2. Implementations **MUST NOT** call `yaml.Unmarshal` on the full compiled YAML solely for template injection purposes when `hasUnsafeExpressionInRunContent` returns `false`.
3. Implementations **SHOULD** share the parsed YAML structure between the template injection validator and any other validators that require it (e.g., schema validation) to avoid redundant unmarshal calls.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. In particular, conformance requires that the pre-flight check never returns `false` when an unsafe expression is present inside a `run:` block (no false negatives), and that the full YAML parse is skipped when the pre-flight returns `false`. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24455330945) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
