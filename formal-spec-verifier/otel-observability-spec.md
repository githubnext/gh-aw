# Formal Notes: otel-observability-spec.md

**Last formalized**: 2026-08-02-15-48-59
**Notation**: TLA+ / Z3-style guard conjunction / F*
**Issue**: pending (created via safe-output; number resolved post-run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1-P15 | (see otel_observability_formal_test.go) | Endpoint normalization, header determinism, Sentry rewrite, if-missing policy, service name, static domain extraction, expression-no-allowlist, top-level header scoping, fan-out ordering, mirror path, empty URL discard, non-Sentry header passthrough, nil/empty headers, invalid if-missing fallback, absent observability |
| P16 | `SecretRefResourceAttributeRejected` | resource-attributes MUST NOT reference secrets.*/vars.* |
| P17 | `CustomAttributesResourceAttributesIndependent` | attributes vs resource-attributes maps stay independent |
| P18 | `MergePrecedenceBaseWinsOverOverride` | base map wins over override map on key collision |
| P19 | `MergeOfEmptyMapsYieldsNil` | merging empty/nil maps yields nil |
| P20 | `MetricResourceCardinalityBound` (stub) | high-cardinality IDs excluded from default metric dimensions |
| P21 | `InstrumentationScopeNaming` (stub) | core scope "gh-aw", gateway scope "gh-aw-mcpg" |

## Key Invariants

- OTLP endpoint normalization always yields an ordered list of entries with non-empty URLs.
- Headers are secrets by default and must never leak into masks-exempt surfaces.
- if-missing defaults to "error"; invalid values fall back silently to "" (treated as error at runtime).
- Resource attributes must not carry `${{ secrets.* }}` / `${{ vars.* }}` expressions.
- Metric providers must not use high-cardinality identifiers (run/job/trace/span/commit/actor/item/conversation IDs) as default dimensions.
- Core vs gateway telemetry must use distinct instrumentation scope names ("gh-aw" vs "gh-aw-mcpg").

## Edge Cases Identified

- Empty/absent `observability.otlp` block (no remote export configured).
- Entries with empty URL discarded with diagnostic.
- Invalid `if-missing` value falls back to default ("error") without crashing.
- Merge of two empty/nil maps must yield nil, not an allocated empty map.
- Sentry-specific header rewriting (`Authorization` → `x-sentry-auth`) must not affect non-Sentry endpoints.

## Notes for Future Runs

- P20 (metric cardinality) and P21 (instrumentation scope naming) currently have **no concrete gh-aw implementation** — tests use stub interfaces. A future run should check whether `pkg/workflow` has grown a metrics-cardinality filter or scope-naming helper, and replace the stubs with real function calls.
- Consider formalizing §6 (Collector vs Direct mode credential isolation), §7 (W3C trace-context propagation across jobs/child workflows), and §16 (Reliability/Failure Handling — flush/shutdown behavior) in a future pass; these sections were read but not yet covered by predicates.
- §17 (Compliance Testing) was not read in this pass — worth reviewing for testable conformance-level assertions (Level 1/2/3) in a subsequent run.
