# Formal Notes: otel-observability-spec.md

**Last formalized**: 2026-08-26-15-58-26
**Notation**: TLA+ / Z3-style guard conjunction / F*
**Issue**: created via safe-output (2026-08-26 run); number resolved post-run

## Predicates

### From 2026-08-02 run (Sections 4-12; file: `otel_observability_formal_test.go`)

| ID | Predicate | Description |
|---|---|---|
| P1-P15 | (see otel_observability_formal_test.go) | Endpoint normalization, header determinism, Sentry rewrite, if-missing policy, service name, static domain extraction, expression-no-allowlist, top-level header scoping, fan-out ordering, mirror path, empty URL discard, non-Sentry header passthrough, nil/empty headers, invalid if-missing fallback, absent observability |
| P16 | `SecretRefResourceAttributeRejected` | resource-attributes MUST NOT reference secrets.*/vars.* |
| P17 | `CustomAttributesResourceAttributesIndependent` | attributes vs resource-attributes maps stay independent |
| P18 | `MergePrecedenceBaseWinsOverOverride` | base map wins over override map on key collision |
| P19 | `MergeOfEmptyMapsYieldsNil` | merging empty/nil maps yields nil |
| P20 | `MetricResourceCardinalityBound` (stub) | high-cardinality IDs excluded from default metric dimensions |
| P21 | `InstrumentationScopeNaming` (stub) | core scope "gh-aw", gateway scope "gh-aw-mcpg" |

### From 2026-08-26 run (Sections 13-16; file: `otel_reliability_formal_test.go`)

| ID | Predicate | Description |
|---|---|---|
| P22 | `OutcomeSeparateTrace` | outcome evaluator delayed >24h MUST NOT extend original workflow trace (§13.1) |
| P23 | `OutcomeSourceCorrelation` | span link OR full attribute triple (run_id/workflow/repo) satisfies source correlation (§13.2) |
| P24 | `OutcomeResultTaxonomy` | `gh-aw.outcome.result` restricted to canonical taxonomy (§13.3) |
| P25 | `MetricDimensionBound` (outcome-specific) | URLs/item_id/run_id MUST NOT be metric dimensions (§13.3/§15.4) |
| P26 | `MirrorPathStable` | local mirror default path is exactly `/tmp/gh-aw/otel.jsonl` (§14.1) |
| P27 | `MirrorWriteBeforeExport` | mirror write occurs before export success is assumed (§14.3) |
| P28 | `MirrorNotTruncatedOnFail` | export failure MUST NOT delete/truncate previously written mirror data (§14.3) |
| P29 | `HeaderRedactionInMirror` | exporter headers/credentials MUST NOT appear in mirrored/artifact records (§15.1) |
| P30 | `ContentDefaultNone` | capture-content defaults to "none"; "full" requires explicit opt-in (§5.6/§15.2) |
| P31 | `PartialFanOutIndependent` | one endpoint failure MUST NOT suppress attempts/successes at sibling endpoints (§16.3) |
| INV2 | `RetryBounded` | retry bounded by max attempts, max elapsed time; permanent failures never retried (§16.2) |
| P32 | `NonFatalExportFailure` | reported workflow result depends only on functional success, not telemetry outcome (§16.1) |
| SAFETY2 | `FailClosedSecrets` | export/fan-out failures and shutdown timeouts recorded as bounded diagnostics, never reported as success, never delete mirror data or expose credentials (Safeguards block) |

## Key Invariants

- OTLP endpoint normalization always yields an ordered list of entries with non-empty URLs.
- Headers are secrets by default and must never leak into masks-exempt surfaces.
- if-missing defaults to "error"; invalid values fall back silently to "" (treated as error at runtime).
- Resource attributes must not carry `${{ secrets.* }}` / `${{ vars.* }}` expressions.
- Metric providers must not use high-cardinality identifiers (run/job/trace/span/commit/actor/item/conversation IDs) as default dimensions.
- Core vs gateway telemetry must use distinct instrumentation scope names ("gh-aw" vs "gh-aw-mcpg").
- Outcome-evaluation spans MUST NOT extend the original workflow trace across long delays; they correlate via span link or explicit source attributes instead.
- The local mirror at `/tmp/gh-aw/otel.jsonl` is write-then-export: writes are unconditional and export failures never delete/truncate mirror data.
- Fan-out across N endpoints is per-endpoint independent; retries are bounded by both attempt count and elapsed time, and permanent failures are never retried.
- Telemetry/export failures are strictly non-fatal to the workflow's functional result (fail-closed for secrets/correctness, fail-open for user-visible success).

## Edge Cases Identified

- Empty/absent `observability.otlp` block (no remote export configured).
- Entries with empty URL discarded with diagnostic.
- Invalid `if-missing` value falls back to default ("error") without crashing.
- Merge of two empty/nil maps must yield nil, not an allocated empty map.
- Sentry-specific header rewriting (`Authorization` → `x-sentry-auth`) must not affect non-Sentry endpoints.
- Outcome evaluation delayed >24h after the source workflow run (must not extend original trace).
- Shutdown flush timing out while mirror records are still pending (must not delete already-written records).
- `capture-content: full` configured without explicit opt-in (must still refuse to capture sensitive content).

## Notes for Future Runs

- P20 (metric cardinality) and P21 (instrumentation scope naming) currently have **no concrete gh-aw implementation** — tests use stub interfaces. A future run should check whether `pkg/workflow` has grown a metrics-cardinality filter or scope-naming helper, and replace the stubs with real function calls.
- Sections 13-16 (Outcome Evaluation, Local Mirrors, Security/Privacy, Reliability) were formalized in the 2026-08-26 run using stub types in `otel_reliability_formal_test.go` (`outcomeSpan`, `mirrorWriter`, `retryPolicy`, `workflowFunctionalResult`, `redactHeaders`) — no concrete Go implementation of outcome-evaluation span emission, mirror writing, or fan-out/retry orchestration exists yet in `pkg/workflow`. A future run should search for real implementations (e.g. in `pkg/cli` or a JS helper under `actions/setup/js/`) and replace stubs with real call sites.
- §17 (Compliance Testing) was read in earlier passes but the Test ID stubs (T-OT-001 through T-OT-007 and beyond) have not been individually mapped to concrete Go tests — worth a dedicated formalization pass to cross-check against `pkg/parser/schema_test.go` and `actions/setup/js/*.test.cjs`.
- §9-12 (Trace Model, Span/Event Contracts, Metrics Contract, Logs Contract) were read but not yet deeply formalized beyond P20/P21 — a good candidate for the next pass on this spec.
