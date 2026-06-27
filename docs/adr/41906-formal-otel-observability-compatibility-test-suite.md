# ADR-41906: Formal OTEL Observability Compatibility Test Suite (v0.4.0)

**Date**: 2026-06-27
**Status**: Draft
**Deciders**: Unknown (automated draft from PR #41906 — review and finalize)

---

### Context

The repository has a formal specification `specs/otel-observability-spec.md` (v0.4.0) that defines 15 predicates for Level 1/2 `observability.otlp` behavior: endpoint form normalization, deterministic header serialization, Sentry `Authorization → x-sentry-auth` rewrite, `if-missing` policy validation, OTEL service-name construction, static domain extraction, expression exclusion from allowlisting, top-level header scoping, fan-out declaration-order preservation, mirror path constants, empty URL entry discard, and nil/empty header normalization. Without explicit test coverage mapped to these predicates, the implementation can silently diverge from the spec as either evolves. The existing `observability_otlp.go` helpers implement these behaviors, but predicate-level regression protection was absent.

### Decision

We will add a dedicated formal test file `pkg/workflow/otel_observability_formal_test.go` that maps each of the 15 predicates in `specs/otel-observability-spec.md` v0.4.0 to an explicit `TestFormal_*` test. Tests assert against the public compiler/runtime behavior of `observability_otlp.go` helpers (e.g., `collectAllOTLPEndpoints`, `normalizeOTLPHeadersForEndpoint`, `otelServiceName`) rather than implementation internals, making the spec a continuously-enforced compatibility contract verified on every CI run.

### Alternatives Considered

#### Alternative 1: Integration/End-to-End Tests

Test the full OTEL pipeline end-to-end (workflow compilation → OTLP emission → collector receipt) rather than testing each predicate in isolation. This would be more realistic but far slower to run and harder to isolate failures to specific predicates. It would also require a running OTEL collector in CI, adding infrastructure complexity. Unit-level predicate tests provide faster feedback and pinpoint exactly which invariant broke.

#### Alternative 2: Property-Based / Fuzz Testing

Use Go's `testing/quick` or fuzz infrastructure to cover the input space for endpoint normalization and header serialization rather than explicit hand-written cases. This would catch edge cases the spec authors didn't enumerate, but would not verify that the specific named invariants from the formal spec are met. Explicit named tests serve as a living checklist against the spec document, which fuzz tests cannot replicate.

### Consequences

#### Positive
- All 15 v0.4.0 predicates from the formal spec are continuously enforced; any regression breaks CI immediately.
- `TestFormal_*` tests serve as executable documentation — future contributors can cross-reference the spec and know exactly which test covers which predicate.
- Tests run at unit speed with no external dependencies, giving fast CI feedback.

#### Negative
- Tests directly call package-internal functions (`collectAllOTLPEndpoints`, `normalizeOTLPHeadersForEndpoint`, etc.), so internal API refactors require corresponding test updates even if observable behavior is unchanged.
- When `specs/otel-observability-spec.md` is updated (v0.5.0, etc.), the test file must be manually updated to reflect new or changed predicates — there is no automated linkage between the spec document and the test file.

#### Neutral
- The formal test file lives in `package workflow` (not `package workflow_test`), giving it access to unexported symbols — this is intentional to allow contract-level assertions without adding a testing-only public API.
- The `determinismTestIterations = 10` constant for `TestFormal_HeaderMapDeterminism` is a pragmatic choice; it is not derived from the formal spec and may need tuning if map iteration order changes are suspected.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
