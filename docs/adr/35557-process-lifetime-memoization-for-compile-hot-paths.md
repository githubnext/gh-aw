# ADR-35557: Process-Lifetime Memoization for Compile Hot Paths

**Date**: 2026-05-28
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

`BenchmarkCompileComplexWorkflow` regressed from 3.06 ms to 4.20 ms (+37%). CPU profiling attributed roughly 26% of CPU time to garbage collection, indicating that the regression was driven by allocation pressure rather than algorithmic cost. Three call sites on the workflow compile path repeatedly recomputed deterministic results from package-level constants on every compile: runtime jsonschema validation of self-generated AWF config JSON in `BuildAWFConfigJSON`, `json.MarshalIndent` of the static `ValidationConfig` map in `GetValidationConfigJSON`, and the deduplicated manifest-files slice in `getAllManifestFiles()`. None of these inputs change across compiles within a process, so each recomputation consumed CPU and produced garbage with no production benefit.

### Decision

We will introduce **process-lifetime memoization** for compile-path computations whose inputs are package-level constants or stable input sets, and **remove runtime schema validation of self-generated JSON** from the compile hot path while preserving schema coverage as a unit test. Specifically: (1) delete the `validateAWFConfigJSON` call in `BuildAWFConfigJSON` and add `TestBuildAWFConfigJSON_SchemaCompliance` to assert schema conformance over representative configurations; (2) cache `GetValidationConfigJSON` output in a `sync.Map` keyed by the sorted, comma-joined enabled-type names; (3) cache the no-extra-files result of `getAllManifestFiles()` behind a `sync.Once`, sharing the underlying computation with the with-extras path via a new `buildBaseManifestFiles()` helper.

### Alternatives Considered

#### Alternative 1: Compilation-scoped (per-compile) memoization

Following the precedent in [ADR-28560](28560-compilation-scoped-memoization-for-permissions-and-domain-computation.md), we could attach a per-compile cache to the compile context and discard it at the end of each compile. This avoids any process-lifetime state and keeps memory tightly scoped. Rejected because the cached values here depend only on package-level constants (`ValidationConfig`, `knownRuntimes`, `securityConfigFiles`), not on per-compile inputs — so the per-compile cache would still recompute identical results across every compile in the same process, leaving the bulk of the regression unaddressed.

#### Alternative 2: Keep runtime schema validation; optimize jsonschema compilation

We could keep `validateAWFConfigJSON` on the hot path and instead memoize the compiled `jsonschema.Schema` object or switch to a lighter validator. Rejected because the validation is structurally redundant — `BuildAWFConfigJSON` self-generates the JSON from a typed Go struct that the schema is derived from, so a runtime check on every compile catches only bugs that a unit test would catch once. Schema compliance belongs in CI, not in every production compile.

#### Alternative 3: `sync.Pool` for marshal buffers

We could keep recomputing the JSON but recycle the underlying byte buffers via `sync.Pool` to reduce allocation pressure. Rejected because it addresses only the byte-allocation slice of the cost while still re-running `json.MarshalIndent` and schema validation each time, leaving CPU work on the table. Memoization eliminates the work entirely for the common case.

### Consequences

#### Positive
- Local benchmark: −10.7% ns/op, −8.5% B/op, −14.7% allocs/op on `BenchmarkCompileComplexWorkflow`.
- Proportional GC reduction; on constrained CI runners where GC dominates wall time, the improvement amplifies beyond the local numbers.
- Aligns with existing memoization precedent in the codebase ([ADR-28557](28557-zero-allocation-fast-path-for-parser-hot-paths.md), [ADR-28560](28560-compilation-scoped-memoization-for-permissions-and-domain-computation.md), [ADR-28744](28744-cache-concurrency-validation-and-toolset-parsing.md)).
- Schema-compliance coverage becomes explicit and auditable as a dedicated test rather than a hidden side effect of compile.

#### Negative
- Schema-compliance coverage shifts from "every call, all configurations" to "fixed test fixtures." A code change that produces a schema-divergent JSON shape not exercised by `TestBuildAWFConfigJSON_SchemaCompliance` would no longer fail at compile time.
- `validationConfigJSONCache` (`sync.Map`) grows for the lifetime of the process — bounded by the number of distinct enabled-type sets, which is small in practice but not zero.
- Process-lifetime caches make benchmarks and tests sensitive to ordering and warm-state; first-call cost differs from steady-state cost, which can complicate microbenchmarks of these functions specifically.

#### Neutral
- `buildBaseManifestFiles()` is now a shared helper between cached and uncached `getAllManifestFiles` paths — a small structural refactor introduced to keep the two code paths in lockstep.
- The cache key for `GetValidationConfigJSON` normalizes ordering via `slices.Sorted`, so callers that supply enabled-types in different orders will share cache entries.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Memoization Eligibility

1. A function on the compile hot path **MAY** introduce process-lifetime memoization (`sync.Once`, `sync.Map`, or equivalent) only if its output is fully determined by package-level constants or by a normalized, finite input key.
2. Memoization keys derived from caller-supplied collections **MUST** be normalized (e.g. sorted) so that callers passing the same logical input in different orders share cache entries.
3. Cached output values **MUST NOT** contain references to caller-owned mutable state; cached slices and maps **SHOULD** be treated as read-only by callers, or **MUST** be defensively copied if returned to mutating call sites.
4. Functions that accept variadic or optional caller inputs **MUST** keep the uncached code path correct for non-empty inputs; the cache **MAY** apply only to the canonical "no-extra-input" case.

### Schema Validation on the Compile Path

1. The compile hot path **MUST NOT** invoke runtime jsonschema validation of JSON that the compiler itself generates from typed Go structs.
2. Schema-compliance coverage for compiler-generated JSON **MUST** be preserved as one or more dedicated unit tests that exercise representative configurations.
3. Such schema-compliance tests **SHOULD** invoke the same validator function the compiler previously used at runtime, so a divergence between the embedded schema and the generated JSON is caught in CI.

### Cache Lifecycle and Boundedness

1. Process-lifetime caches keyed by caller input **SHOULD** be bounded in expected cardinality (small, finite set of distinct keys); unbounded growth driven by user input is **NOT RECOMMENDED**.
2. Caches **MUST NOT** persist values that depend on per-compile, per-workflow, or per-request mutable state — such values **MUST** instead use compilation-scoped memoization as described in [ADR-28560](28560-compilation-scoped-memoization-for-permissions-and-domain-computation.md).
3. Cached computations **MUST** remain deterministic with respect to their declared inputs; nondeterministic inputs (clock, random, environment) **MUST NOT** participate in cache keys without explicit invalidation.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26600090198) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
