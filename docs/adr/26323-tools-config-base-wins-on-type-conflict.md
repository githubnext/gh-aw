# ADR-26323: Tools Configuration Merge — Base Workflow Wins on Type Conflict

**Date**: 2026-04-14
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw workflow compiler supports importing shared workflows. Both the main (base) workflow and imported shared workflows can declare `tools` configuration. When the compiler merges these configurations, it must decide which value takes precedence when the same key appears in both. The documented principle is "the main workflow overrides imports." Before this fix, `MergeTools` in `pkg/parser/tools_merger.go` had a type-mismatch fallback that overwrote the base value with the additional (import) value, silently violating that principle. A concrete symptom: setting `tools.github: false` in a main workflow had no effect if any imported shared workflow declared `tools.github` as a map (e.g., `tools.github: mode: remote`), leaving the GitHub MCP server enabled against the user's explicit intent.

### Decision

We will make the base (main workflow) value win unconditionally for all non-array, non-map conflicts in `MergeTools`, including type mismatches. When both values are scalars of different types — such as `bool` vs. `map` — the base value is kept and the import value is discarded. Arrays are always union-merged (deduplicated). Maps are recursively merged. This extends and makes uniform the existing "workflow overrides imports" principle for every scalar conflict, not just same-type conflicts.

### Alternatives Considered

#### Alternative 1: Return an error on type mismatch

Treat a type mismatch between base and additional as a configuration conflict and fail compilation with a descriptive error, requiring the author to resolve the inconsistency explicitly.

This was considered because it makes conflicts visible rather than silently ignoring one side. It was rejected because it would be a breaking change — existing workflows that import shared configs with overlapping keys of different types would start failing — and because the correct resolution is already well-defined by the documented "base wins" principle.

#### Alternative 2: Log a warning and keep the existing (pre-fix) behaviour

Emit a warning when a type mismatch is encountered but continue to let the import value override the base, matching the old behaviour.

This was rejected because it preserves the incorrect behaviour: an explicit `tools.github: false` in the main workflow would still be silently overridden by an import. The documented contract — that main workflows override imports — would remain broken.

#### Alternative 3: Require explicit `override:` syntax for scalar conflicts

Introduce a dedicated `override:` field or a special sentinel value in frontmatter so that the author must consciously express "this value should win even if the type differs."

This was considered as a more principled long-term design but rejected for this fix because it requires schema changes, parser updates, and documentation work that are disproportionate to the scope of the bug. The "base always wins" rule is already documented and consistent with all other merge behaviour.

### Consequences

#### Positive
- `tools.github: false` (and equivalent scalar disable values) in a main workflow now reliably prevents imported configurations from re-enabling that tool.
- The `MergeTools` function behaves consistently: base wins for all scalar conflicts, regardless of type, matching the documented contract.
- No behaviour change for same-type scalar overrides (those were already base-wins in practice, since `result[key]` was set from base before the loop) — only the type-mismatch branch changes.

#### Negative
- Any workflow that previously relied on an import overriding a scalar value in the base via a type mismatch will now silently behave differently. This was always the intended behaviour, but any workflow depending on the bug would now change.
- The fix is a silent behaviour change with no migration path; authors won't receive an error or warning if their import's value is now discarded.

#### Neutral
- The fix is a single-line removal in `tools_merger.go` (`result[key] = newValue` is removed from the type-mismatch branch).
- Two new test files (`frontmatter_merge_test.go`, `importable_tools_test.go`) document the expected merge semantics and serve as regression guards.
- Recursive map merging and array union-merge behaviour are unchanged.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Tools Configuration Merge Semantics

1. Implementations **MUST** give the base (main workflow) value precedence over the additional (imported workflow) value when merging `tools` configuration and a conflict exists for the same key.
2. Implementations **MUST NOT** overwrite a base scalar value with an additional value when the two values are of different types (e.g., `bool` vs. `map`).
3. Implementations **MUST** union-merge array values (with deduplication) when both the base and additional values for a key are arrays.
4. Implementations **MUST** recursively merge map values when both the base and additional values for a key are maps, applying the same base-wins precedence rule at every level of nesting.
5. Implementations **MUST** add keys that are present in the additional configuration but absent from the base configuration without modification.

### Conflict Resolution

1. Implementations **MUST** treat a type mismatch between a base value and an additional value for the same key as a base-wins conflict, retaining the base value and discarding the additional value.
2. Implementations **SHOULD NOT** emit an error or warning for type-mismatch conflicts; silent base-wins resolution is the intended behaviour.
3. Implementations **MAY** log a debug-level message when a type-mismatch conflict is silently resolved, to aid diagnostics.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement — in particular allowing an import to override a base scalar value through a type mismatch — constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24428894483) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
