# ADR-49809: Formalize OTLP Observability Spec with Executable Predicates P16–P21

**Date**: 2026-08-02
**Status**: Accepted

---

### Context

The `gh-aw` workflow runtime has an OTLP observability integration (`pkg/workflow/observability_otlp.go`) that enforces several security-critical and semantic invariants: resource-attribute values must not reference `secrets.*` or `vars.*` expressions, span attributes and resource attributes are parsed from disjoint frontmatter keys, and the `mergeOTLPStringMaps` utility follows base-wins precedence with a nil-as-sentinel contract for empty inputs. Additionally, the system is expected to enforce metric cardinality bounds and distinct instrumentation scope names in future work. These invariants were previously verified only through informal tests or not at all, making it difficult to audit, communicate, or regress-test the spec. An existing formal predicate convention (P1–P15) in `otel_observability_formal_test.go` already establishes a numbered, property-named test style that serves as a machine-verifiable living specification.

### Decision

We will extend the formal OTLP observability test suite by adding predicates P16–P21 to `pkg/workflow/otel_observability_formal_test.go`. P16–P19 test concrete behaviors already present in `observability_otlp.go` (secret-ref rejection, attribute independence, merge precedence, nil-as-sentinel). P20 and P21 are forward-looking pending predicates: each test function immediately calls `t.Skip("pending: ...")` with an explicit message explaining what production implementation is required before they can exercise real behavior. The stub interface types (`metricAttributeRegistry`, `instrumentationScopeResolver`) that were previously defined alongside test-only implementations have been removed; the expected contract is instead described in the pending-test comment. This avoids the tautological anti-pattern of a test that constructs its own oracle and validates against that same oracle. When production implementations of the cardinality filter and instrumentation-scope resolver land in `pkg/workflow`, the `t.Skip` calls will be replaced with assertions against those implementations.

### Alternatives Considered

#### Alternative 1: Informal Go unit tests without predicate naming

Add standard `TestXxx` functions without the `P16`-style numbering convention. This was rejected because the numbered predicate convention ties directly to the prose specification (`specs/otel-observability-spec.md`) and makes it easy to cross-reference which behaviors are covered. Informal tests provide the same runtime coverage but lose the self-documenting, spec-linked quality.

#### Alternative 2: Documentation-only approach in `specs/otel-observability-spec.md`

Write the invariants as prose in the specification document without adding executable tests. This was rejected because documentation drifts from implementation over time; executable predicates remain accurate by construction — if the implementation drifts, the test fails. A prose-only spec cannot prevent regressions.

#### Alternative 3: Use Go's built-in fuzzing (`go test -fuzz`) for attribute validation

Apply fuzz testing to `validateOTLPResourceAttributes` and `mergeOTLPStringMaps` to explore the input space rather than using fixed test cases. This was not chosen because the existing P1–P15 suite uses deterministic scenario-based tests, and consistency with the established testing pattern was preferred. Fuzzing could complement the formal predicates in a future iteration.

### Consequences

#### Positive
- P16–P19 are backed by concrete implementations; regressions in security-critical behaviors (secret rejection, attribute isolation, merge semantics) will be caught immediately by CI.
- P20 and P21 establish interface contracts (`metricAttributeRegistry`, `instrumentationScopeResolver`) before the production implementations exist, enabling interface-first design and reducing future coupling.
- The numbered predicate style makes it straightforward to audit which invariants from `specs/otel-observability-spec.md` are machine-verified vs. not yet covered.

#### Negative
- P20 and P21 are currently skipped (`t.Skip("pending: ...")`); they do not exercise real production code until the cardinality filter and instrumentation-scope resolver are implemented.
- When those implementations land, contributors must remove the `t.Skip` calls and supply real assertions; this is enforced only by code-review discipline, not by tooling.

#### Neutral
- The formal test suite now spans P1–P21 across a single file; continued growth will require either continued sequential numbering or a decision to split the file by concern.
- Future contributors must follow the P-numbered predicate naming convention when adding invariants; this convention is not enforced by tooling, only by code-review discipline.

---

*ADR created by [adr-writer agent], finalized in [PR #49809](https://github.com/github/gh-aw/pull/49809).*
