# ADR-25821: Rust-Style Source Context and Plain-English Synthesis for Compiler Errors

**Date**: 2026-04-11
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw workflow compiler produces error messages when a workflow file contains invalid configuration. Before this decision, two distinct quality problems coexisted. First, when the `engine:` field received a value of the wrong type (e.g., an integer instead of a string), the compiler surfaced raw JSON Schema jargon such as `'oneOf' failed, none matched: got number, want string; got number, want object`, giving the user no actionable guidance. Second, when a valid-type but unrecognised engine name was used (e.g., a typo like `copiilot`), the error message correctly identified the problem but omitted source-file context lines, making it harder to pinpoint the exact location. The two code paths had inconsistent output quality, and the gap was widest precisely when users were most confused.

### Decision

We will adopt Rust-style compiler error rendering as the standard for field-level validation errors in the workflow compiler: errors that can be localised to a specific source line will include ±3 lines of source context to allow the user to see the offending code in-line. Simultaneously, we will introduce plain-English synthesis for JSON Schema `oneOf` type-conflict failures: when every branch of a `oneOf` constraint fails with a type mismatch, the compiler extracts the actual and expected types and produces a sentence such as `expected string or object, got number`. For well-known fields (currently `/engine`), a field-specific hint table appends a list of valid values and a usage example, creating output comparable to modern language compilers (Rust, Go 1.20+).

### Alternatives Considered

#### Alternative 1: Generic "wrong type" message without field hints

The simplest fix was to replace JSON Schema jargon with a static message like `"invalid value: expected a string or object"` for all `oneOf` type failures. This was considered because it requires no per-field maintenance. It was rejected because it omits valid values, which is the most actionable piece of information for the user — knowing that `claude`, `codex`, `copilot`, and `gemini` are accepted is more useful than knowing the abstract type.

#### Alternative 2: Runtime engine list lookup from the catalog

An alternative to the static hint table was to look up the list of valid engine names dynamically from `NewEngineCatalog()` at error-reporting time. This would have made the hint automatically accurate if new built-in engines were added. It was not chosen because error formatting is in `pkg/parser` (a lower-level package), while the engine catalog lives in a higher-level layer; pulling the catalog reference into the parser would create an undesirable dependency. The static list is a deliberate trade-off: it requires a manual update when built-in engines change, but it preserves the package boundary.

#### Alternative 3: Position-only error (status quo for engine name errors)

The existing `formatCompilerErrorWithPosition` path was already used for engine-name typos (e.g., `copiilot`). Keeping that path and only fixing the type-conflict message was considered as a minimal change. It was rejected because source context lines are low-cost to add (the file content is already in memory at this call site) and significantly improve diagnostic value, aligning the output with user expectations set by modern compilers.

### Consequences

#### Positive
- Type-conflict errors for `engine:` are now actionable without requiring the user to look up documentation.
- Engine-name typos now show a Rust-style source snippet with a column pointer, matching output quality of schema-validation errors.
- The `isTypeConflictLine` predicate is now precise: it rejects constraint-violation lines (e.g., `minItems: got 0, want 1`) that were previously false-positived, reducing noise in other `oneOf` error paths.

#### Negative
- The `knownOneOfFieldHints` table in `pkg/parser/schema_errors.go` is a static list of field paths and valid values. It will silently become stale if built-in engines are added or removed without also updating the table.
- `readSourceContextLines` always returns a fixed 7-line window (±3). Errors near the start or end of a file receive padding with empty strings, which requires downstream rendering logic to tolerate empty lines gracefully.

#### Neutral
- Rust-style rendering requires the caller (currently `compiler_orchestrator_workflow.go`) to pass pre-loaded file content to `readSourceContextLines`. This makes the call site slightly more verbose but keeps the I/O concern at the orchestration layer.
- The `console.CompilerError.Context` field must already be populated by other code paths; this change adds a second call site that exercises it.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Field-Level Error Rendering

1. When a compiler error is localised to a specific source line and the file content is available in memory at the call site, the implementation **MUST** include source context lines in the error using `formatCompilerErrorWithContext`.
2. Implementations **MUST NOT** call `formatCompilerErrorWithPosition` for errors where file content is already loaded; `formatCompilerErrorWithContext` **MUST** be used instead.
3. Source context **MUST** span a window of ±3 lines around the error line (7 lines total), padded with empty strings when the window extends before the start of the file.
4. Implementations **SHOULD** pass `nil` as the `cause` argument to `formatCompilerErrorWithContext` for pure validation errors that have no underlying Go error to wrap.

### oneOf Type-Conflict Error Synthesis

1. When all sub-errors of a `oneOf` constraint are type conflicts (i.e., `cleanOneOfMessage` produces an empty `meaningful` list), implementations **MUST** call `synthesizeOneOfTypeConflictMessage` rather than returning the raw jargon string.
2. `synthesizeOneOfTypeConflictMessage` **MUST** produce a message of the form `"expected T1 or T2, got G"` where `T1`, `T2` are the distinct expected JSON Schema type names and `G` is the actual type.
3. When the JSON Schema path of the failing field matches a key in `knownOneOfFieldHints`, the synthesized message **MUST** append the corresponding hint text.
4. `knownOneOfFieldHints` **MUST** be updated whenever the set of built-in engines in `NewEngineCatalog` changes.
5. Implementations **MUST NOT** include field-specific hints for paths not present in `knownOneOfFieldHints`; unknown fields **SHOULD** receive the generic type-mismatch message only.

### Type-Conflict Line Detection

1. `isTypeConflictLine` **MUST** validate that both the "got" and "want" tokens in a `"got X, want Y"` pattern are valid JSON Schema type names (`string`, `object`, `array`, `number`, `integer`, `boolean`, `null`) before classifying the line as a type conflict.
2. Implementations **MUST NOT** classify constraint-violation lines (e.g., `"minItems: got 0, want 1"`) as type conflicts.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24285718804) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
