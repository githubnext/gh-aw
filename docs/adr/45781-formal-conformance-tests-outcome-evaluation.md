# ADR-45781: Formal Conformance Test Suite for Outcome Evaluation Engine

**Date**: 2026-07-15
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `safe-output-outcome-evaluation.md` spec defines 12 behavioral predicates (P1–P12) that govern how the outcome evaluation engine classifies safe-output results (accepted, rejected, pending, lifecycle, etc.). These predicates existed only as informal English prose and TLA+/F*/Z3 notation in the spec, with no executable machine-checkable tests to verify the production code conformed to them. Two specific gaps blocked full coverage: `OutcomeLifecycleClose` was the sixth spec-defined observable outcome but was absent as a typed Go constant, and `evalCloseSticky` called `ghAPIGet` directly rather than through a package-level variable, preventing unit-test injection.

### Decision

We will encode all 12 spec predicates as formal-notation specifications (TLA+ invariants, F* pre/post contracts, and Z3/SMT-LIB arithmetic bounds) in the spec document, and as an executable Go conformance test suite (`pkg/cli/outcome_eval_formal_test.go`). Two production code changes enable full coverage: adding `OutcomeLifecycleClose OutcomeResult = "lifecycle_close"` to close the typed-constant gap, and extracting `closeStickyGHAPIGet = ghAPIGet` as a package-level variable in `outcome_eval_generic.go` to make `evalCloseSticky` mockable, consistent with the existing `genericOutcomeGHAPIGet` / `outcomeUpdateGHAPIGet` pattern.

### Alternatives Considered

#### Alternative 1: Integration Tests Only

Test the evaluation engine against a live GitHub API instead of unit tests. This would validate end-to-end behavior but requires live credentials and infrastructure, cannot run in the standard CI unit-test suite, and is far slower than unit tests — making it unsuitable as the primary conformance mechanism for spec predicates.

#### Alternative 2: Informal Prose Tests Without Formal Notation Cross-References

Write tests using plain Go without embedding formal-notation cross-references (TLA+/F*/Z3 identifiers). Easier to write and more accessible to reviewers unfamiliar with formal methods, but leaves the connection between spec predicates and test code implicit — making it harder to audit coverage gaps and to verify that a test actually exercises the stated invariant.

#### Alternative 3: Keep `evalCloseSticky` Non-Mockable (Skip P2/P9/P11/P12 Unit Tests)

Avoid the `closeStickyGHAPIGet` extraction and rely on higher-level integration tests for the close-sticky code path. This would preserve the production code's simplicity but would leave predicates P2, P9, P11, and P12-Class-C unverifiable without live GitHub API access.

### Consequences

#### Positive
- All 12 spec predicates (P1–P12) now have executable, machine-checkable conformance tests runnable with `go test ./pkg/cli/` — no special flags or infrastructure required.
- `closeStickyGHAPIGet` extraction follows the established `*GHAPIGet` variable pattern, making it consistent and idiomatic within the codebase.
- Spec predicates and test functions are explicitly cross-referenced by identifier (e.g., `P9`), making coverage audits mechanical.

#### Negative
- `OutcomeLifecycleClose` is added as a typed constant and tested in P1/P6, but the corresponding `OutcomeStatusLifecycle` status constant is deferred — the test explicitly documents this with a TODO, leaving a partial implementation.
- Formal notation (TLA+, F*, Z3/SMT-LIB) appears only in spec prose and Go test comments, not as verified-by-tooling artifacts. The guarantees expressed in those notations are not mechanically checked by a TLA+ model checker or F* verifier.
- Adding a package-level mutable `closeStickyGHAPIGet` variable introduces test-induced global mutable state, a known Go tradeoff that relies on `t.Cleanup` for isolation.

#### Neutral
- The 12 tests carry `//go:build !integration`, so they run in the default unit-test suite and are excluded from integration test runs — no change to the integration test surface.
- The spec additions (Formal Model, Behavioral Coverage Map, Generated Test Suite sections) are append-only and do not modify any existing spec normative content.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
