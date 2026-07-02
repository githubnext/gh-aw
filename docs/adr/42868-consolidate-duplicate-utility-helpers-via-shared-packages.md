# ADR-42868: Consolidate Duplicate Utility Helpers via Shared Packages

**Date**: 2026-07-02
**Status**: Draft
**Deciders**: Unknown

---

### Context

Multiple packages in this codebase (`pkg/workflow`, `pkg/cli`, `pkg/console`, `pkg/intent`, `pkg/parser`) had developed near-identical local implementations of common utility functions: JSONL log parsing for AI engine responses, byte-unit scaling, sorted-key extraction from maps, repo slug splitting, and string-distance helpers. The log parsing duplication was particularly pronounced — `GeminiEngine.ParseLogMetrics` and `AntigravityEngine.ParseLogMetrics` each contained ~85 lines of effectively identical logic, differing only in an engine name string. Over-engine override methods (`GetLogFileForParsing`, `GetDefaultDetectionModel`) also existed on both engines despite matching `BaseEngine` defaults exactly. Local wrappers like `splitOwnerRepo`, `sortedRemainingPermissionKeys`, and `cloneStrings` were single-call pass-throughs to existing utility packages, adding indirection with no value.

### Decision

We will eliminate confirmed duplicates by extracting a shared `parseStatsJSONLMetrics` helper (parameterized by engine name and logger) in `pkg/workflow/stats_jsonl_logs.go`, and by replacing all local single-purpose wrapper functions and manual map-to-sorted-slice idioms with direct calls to the existing shared utility packages (`sliceutil`, `repoutil`, `stringutil`, `slices`). Engine-specific overrides that merely echo `BaseEngine` behavior will be removed.

### Alternatives Considered

#### Alternative 1: Keep Per-Package Copies

Each package maintains its own implementation, accepting intentional duplication in exchange for full package isolation. Changes to the shared logic would not ripple across packages.

This was rejected because the duplicates were not intentionally divergent — they were accidental copies. Keeping them increases the chance that a bug fix applied in one place is missed in another. The packages involved (`workflow`, `cli`, `console`) already share other utilities, so the isolation argument does not hold in practice.

#### Alternative 2: Abstract via Interface

Define a shared `LogParser` interface that each engine implements independently, so engines remain decoupled even if their current implementations happen to be identical.

This was rejected because the engines' log format is already the same (stats-JSONL), and an interface would require callers to depend on an abstraction rather than the concrete behavior. The shared helper is an internal package function, not a public contract, so adding an interface layer would introduce complexity with no consumer benefit at this time.

### Consequences

#### Positive
- Bug fixes and improvements to the shared log-parsing logic apply to all engines simultaneously, reducing the risk of silent divergence.
- Removal of pass-through wrappers makes package-level dependencies explicit: callers now import the authoritative utility package directly.
- Reduced overall line count and cognitive surface area across the affected packages.

#### Negative
- `pkg/workflow/antigravity_logs.go` and `pkg/workflow/gemini_logs.go` now depend on the new internal `parseStatsJSONLMetrics` helper; a future engine that genuinely needs different log-parsing behavior must either extend the helper or add its own implementation.
- Removing `GetLogFileForParsing` and `GetDefaultDetectionModel` overrides from Gemini and Antigravity engines silently relies on `BaseEngine` defaults being correct for those engines going forward — this invariant is not tested.

#### Neutral
- The public API surface of affected packages is unchanged; this is an internal refactoring only.
- Tests added for `GeminiEngine.ParseLogMetrics` provide coverage that previously only existed for the Antigravity parser.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
