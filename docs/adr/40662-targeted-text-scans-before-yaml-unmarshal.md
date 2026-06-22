# ADR-40662: Use Targeted Text Scans to Gate yaml.Unmarshal in Template-Injection Validation

**Date**: 2026-06-21
**Status**: Accepted

## Context

`BenchmarkCompileComplexWorkflow` regressed ~320% (from ~3ms/op to ~12.7ms/op). CPU profiling traced the cost to `validateTemplateInjection` Path B — the code path taken whenever `skipValidation=true`, which is the default in `NewCompiler()`. That path used `hasAnyExpressionInRunContent`, a pre-check that matched **any** `${{ }}` expression in a `run:` block, including compiler-owned references such as `${{ runner.temp }}`. Because compiled workflows almost always contain such safe expressions, the pre-check effectively always returned true and unconditionally triggered a full `yaml.Unmarshal` of the ~1200-line compiled YAML on every compilation, even when no injection or regression violation was possible.

## Decision

We will replace the broad any-expression pre-check in `validateTemplateInjection` Path B with two targeted text scans that map one-to-one onto the two downstream sub-validators, and only call `yaml.Unmarshal` when at least one scan signals a potential issue. `needsUnsafeCheck` uses `hasExpressionInRunContent(UnsafeContextPattern)` to detect user-controlled contexts (template-injection risk), and `needsDisallowedCheck` uses the new `hasNonAllowedExpressionInRunContent` to detect expressions outside the compiler-owned allow-list (`allowedRunScriptExpressionRegex`). Both scans share the same raw-YAML `run:` walker, including flow-style and quoted-key support, so they stay aligned with each other and with the parsed-YAML validators. For well-formed workflows both scans return false and the YAML re-parse is skipped entirely.

## Alternatives Considered

### Alternative 1: Reuse the schema-validation parsed tree in Path B
We could enable schema validation (or share its parsed `map[string]any`) so Path B never re-parses. This was rejected because `skipValidation=true` is the intentional default fast path; forcing a parse there would reintroduce the very `yaml.Unmarshal` cost this change eliminates and change compiler semantics for callers that deliberately skip schema checks.

### Alternative 2: Narrow `hasAnyExpressionInRunContent` to ignore allowed expressions only
We could keep a single combined pre-check that skips allow-listed expressions but still gate both validators together. This was rejected because the two validators have genuinely different trigger conditions (unsafe-context vs. non-allowed-expression); collapsing them would run one validator unnecessarily whenever the other's condition is met, and splitting the scans keeps each sub-validator's cost proportional to its own risk signal.

## Consequences

### Positive
- `BenchmarkCompileComplexWorkflow` drops from ~6.8ms/op to ~1.64ms/op and allocations from 98,282 to 10,894 per op.
- `yaml.Unmarshal` is skipped entirely in the common case (well-formed compiled workflows with only compiler-owned expressions).
- The two scans now correspond directly to the two sub-validators, making the trigger logic easier to reason about.

### Negative
- The scan logic is more complex: the shared raw-YAML `run:` walker must stay consistent with the YAML parser's view of the same structure, including block scalars, flow-style mappings, and quoted keys.
- Two regexes (`UnsafeContextPattern`, `allowedRunScriptExpressionRegex`) plus block detection now encode security-relevant gating; a bug that wrongly returns false would silently skip a security validator.

### Neutral
- Behavior is unchanged when a violation is actually present — the YAML is still parsed and the same validators run.
- `allowedRunScriptExpressionRegex` becomes part of the performance-critical fast path; extending the compiler's allow-list now also affects which expressions can skip the parse.
