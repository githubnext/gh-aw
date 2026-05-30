# ADR-28744: Cache Concurrency Validation and Toolset Parsing in WorkflowData

**Date**: 2026-04-27
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

`WorkflowData` is the central data carrier for the compiler pipeline. A prior optimization (ADR-28560) cached permissions parsing and allowed-domain computation in `applyDefaults`, recovering a 21% regression in `BenchmarkCompileSimpleWorkflow`. However, `BenchmarkValidation` was still 119.3% above its historical baseline (250 µs vs. 114 µs) because two additional expensive operations remained uncached in the hot `validateWorkflowData` loop: (1) `validateConcurrencyGroupExpression`, which runs the full `ExpressionParser` (regex scan + tokenize + parse pass) on the auto-generated workflow-level concurrency expression, and (2) `ParseGitHubToolsets`, which was called twice per iteration — once inside `ValidatePermissions` and again in `validateToolConfiguration`. Together these accounted for 47% of all allocations per benchmark iteration.

### Decision

We will extend the `WorkflowData` eager-caching pattern established in ADR-28560 to cover concurrency group expression validation and GitHub toolset parsing. In `applyDefaults`, after all mutations are complete, the defer block will: (a) pre-validate the concurrency group expression via `validateConcurrencyGroupExpression` and store the result in `CachedConcurrencyGroupExprErr`, using a boolean sentinel `CachedConcurrencyGroupExprSet` to distinguish a valid `nil` result from "not yet computed"; and (b) call `ParseGitHubToolsets` once and store the result in `CachedParsedToolsets`. `ValidatePermissions` will accept an optional variadic `parsedToolsets ...[]string` parameter so callers can pass the pre-parsed slice directly, eliminating the redundant parse inside that function. Both call sites add a fallback code path that live-computes the value when the cache is not populated, preserving compatibility with `WorkflowData` instances created outside `ParseWorkflowFile`.

### Alternatives Considered

#### Alternative 1: Lazy nil-check memoization (same pattern as `CachedPermissions`)

ADR-28560 used a nil check (`if data.CachedPermissions == nil`) as the sentinel. The same approach would avoid a dedicated boolean flag. This was not chosen because the validation result for a _valid_ concurrency expression is `nil` (no error), making a nil check ambiguous: a nil `CachedConcurrencyGroupExprErr` could mean either "not yet computed" or "computed and valid." A separate boolean `CachedConcurrencyGroupExprSet` is necessary to make the distinction explicit and avoid silently skipping validation when the cache is unpopulated.

#### Alternative 2: Thread pre-computed values through function parameters

Passing the pre-parsed toolsets and validation result as explicit parameters to `validateWorkflowData` and its callees would avoid adding new cache fields to `WorkflowData`. This was not chosen because it requires cascading signature changes across multiple callers, increases call-site complexity, and departs from the established pattern in this codebase of using `WorkflowData` as the compilation-scoped state carrier for derived values.

### Consequences

#### Positive
- `BenchmarkValidation` drops from ~250 µs to ~2,390 ns/op — approximately 3× faster — and falls well below the 114 µs historical baseline.
- Allocations per operation fall from 40 to 10 (75% reduction); memory per operation falls from 2,080 B to 552 B (73% reduction).
- The variadic `parsedToolsets ...[]string` parameter is fully backward-compatible: existing callers of `ValidatePermissions` require no changes.

#### Negative
- `WorkflowData` gains two more cache fields (`CachedConcurrencyGroupExprErr`, `CachedConcurrencyGroupExprSet`) plus `CachedParsedToolsets`, expanding the implicit contract that `applyDefaults` must run before any downstream validator reads these fields.
- The boolean sentinel pattern (`CachedConcurrencyGroupExprSet`) is inconsistent with the empty-string sentinel used by `CachedAllowedDomainsStr` (ADR-28560), creating two different caching idioms on the same struct.
- `ValidatePermissions` gains a variadic parameter; while backward-compatible, it increases the function's surface area and makes call-site intent less obvious without documentation.

#### Neutral
- Both new caching sites add a fallback live-compute path for `WorkflowData` instances created without going through `ParseWorkflowFile`. These fallback paths must be kept in sync with the primary computation logic.
- This PR narrows the scope of `validateConcurrencyGroupExpression` as called from `validateToolConfiguration`: it now only runs as a fallback, never as the primary path for `WorkflowData` produced by `ParseWorkflowFile`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Cache Population

1. The `applyDefaults` defer block **MUST** populate `CachedConcurrencyGroupExprErr` and set `CachedConcurrencyGroupExprSet = true` after `ConcurrencyGroupExpr` has been extracted, when `ConcurrencyGroupExpr` is non-empty.
2. The `applyDefaults` defer block **MUST** populate `CachedParsedToolsets` when `data.ParsedTools != nil && data.ParsedTools.GitHub != nil`.
3. Cache fields **MUST** be populated in the `applyDefaults` defer block, after all mutations to `WorkflowData` are complete, so that downstream validators see stable, final values.

### Cache Consumption

1. `validateToolConfiguration` **MUST** use `CachedConcurrencyGroupExprErr` and `CachedConcurrencyGroupExprSet` when they are available, and **MUST NOT** call `validateConcurrencyGroupExpression` directly when `CachedConcurrencyGroupExprSet` is `true`.
2. `validateToolConfiguration` **MUST** fall back to calling `validateConcurrencyGroupExpression` directly when `CachedConcurrencyGroupExprSet` is `false`, to preserve correctness for `WorkflowData` created outside `ParseWorkflowFile`.
3. `validateToolConfiguration` **MUST** use `CachedParsedToolsets` when it is non-nil, and **MUST** fall back to `ParseGitHubToolsets` when `CachedParsedToolsets` is nil.
4. `ValidatePermissions` **MUST** use the first element of the variadic `parsedToolsets` argument when it is non-nil, and **MUST** fall back to calling `ParseGitHubToolsets(githubTool.GetToolsets())` when no pre-parsed slice is provided or the slice is nil.

### Sentinel Convention

1. `CachedConcurrencyGroupExprSet` **MUST** be `true` if and only if `validateConcurrencyGroupExpression` has been called and its result stored in `CachedConcurrencyGroupExprErr`.
2. Implementations **MUST NOT** interpret a nil `CachedConcurrencyGroupExprErr` as evidence that validation has been performed; `CachedConcurrencyGroupExprSet` **MUST** be checked first.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: (1) `applyDefaults` populates all three cache fields before any downstream validator runs; (2) `validateToolConfiguration` short-circuits to cached values when available and falls back to live computation otherwise; (3) `ValidatePermissions` respects the variadic toolsets parameter; and (4) the boolean sentinel `CachedConcurrencyGroupExprSet` is always checked before reading `CachedConcurrencyGroupExprErr`.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25006605654) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
