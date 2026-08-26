# ADR-56076: Formal Predicates for OTel Outcome Evaluation, Mirrors, Security, and Reliability (§13-16)

**Date**: 2026-08-26
**Status**: Draft
**Deciders**: Unknown (automated draft from issue #56076 — review and finalize)

---

### Context

`specs/otel-observability-spec.md` (v0.4.0) Sections 13-16 define normative MUST/SHOULD behavior for outcome-evaluation trace correlation, the `/tmp/gh-aw/otel.jsonl` local mirror's durability guarantees, header/content redaction defaults, and bounded-retry/fail-closed reliability. The prior formalization pass (`pkg/workflow/otel_observability_formal_test.go`, ADR-49809) covered Sections 4-12 and established a precedent for pending predicates: `t.Skip("pending: ...")` with an explicit rationale, rather than a stub interface validated against its own test-only implementation, which would be tautological.

No outcome-evaluation span emitter or mirror writer exists in `pkg/workflow`. The mirror writer and fan-out/retry orchestration are implemented in JavaScript (`actions/setup/js/send_otlp_span.cjs`), and the outcome-status taxonomy is implemented in `pkg/cli` (`OutcomeStatus`, covered by `pkg/cli/outcome_eval_formal_test.go`), which `pkg/workflow` cannot import without creating an import cycle (`pkg/cli` already imports `pkg/workflow`).

### Decision

We will add `pkg/workflow/otel_reliability_formal_test.go` mapping the twelve predicates from the issue's behavioral coverage map (P1-P11, INV1) to `TestFormal_*` functions. `TestFormal_MirrorPathIsStable` (P4) is asserted directly against the real `pkg/constants` mirror-path constants, matching the existing `TestFormal_MirrorPathConstant` coverage in the Section 4-12 suite. Every other predicate currently lacks a `pkg/workflow` call site to assert against, so each is expressed as a skipped test with an explicit pending rationale identifying where the real implementation lives today (JavaScript mirror/retry code, or `pkg/cli` outcome taxonomy) and what must land in `pkg/workflow` before the skip can be replaced with real assertions.

### Alternatives Considered

#### Alternative 1: Stub interfaces validated against test-only implementations

Define stub types (`outcomeSpan`, `mirrorWriter`, `retryPolicy`, etc.) as suggested by the issue and assert the test suite's own stub behavior. This was rejected because it is tautological — the test would always pass regardless of the eventual real implementation, providing no regression protection and creating false confidence in coverage. This mirrors the same rejection reasoning already recorded in ADR-49809 for P20/P21.

#### Alternative 2: Import pkg/cli's OutcomeStatus from pkg/workflow

Reference the real outcome taxonomy constants directly for P3. This was rejected because `pkg/cli` imports `pkg/workflow`, so the reverse import would create a package cycle; the taxonomy invariant is already covered independently by `pkg/cli/outcome_eval_formal_test.go`.

### Consequences

#### Positive
- The behavioral coverage map from issue #56076 is fully represented as named, spec-cross-referenced `TestFormal_*` functions, with clear pointers to the real implementation location for each pending predicate.
- No tautological stub-oracle tests are introduced, keeping the formal suite's pass/fail signal meaningful.
- `TestFormal_MirrorPathIsStable` gives immediate regression protection for the one predicate with a real `pkg/workflow` call site.

#### Negative
- Eleven of twelve predicates are currently skipped and do not exercise real production code; regressions in the JavaScript mirror/retry/redaction logic or in `pkg/cli` outcome handling are not caught by this file.
- When a Go outcome-evaluation span emitter, mirror writer, or retry/fan-out orchestrator lands in `pkg/workflow`, contributors must manually replace each `t.Skip` with real assertions.

#### Neutral
- The file lives in `package workflow` (not `_test` suffix package) for consistency with the existing `otel_observability_formal_test.go` suite, though none of the pending tests currently need unexported access.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
