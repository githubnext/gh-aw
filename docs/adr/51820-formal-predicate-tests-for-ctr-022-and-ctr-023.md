# ADR-51820: Formal Predicate-Level Unit Tests for CTR-022 and CTR-023

**Date**: 2026-08-10
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The GitHub Actions Compiler Threat Detection Specification (v1.0.20) introduced two new rules: CTR-022 (Git Subprocess Argument Injection) and CTR-023 (Bash Command Allowlist Illusion). Both rules have concrete, exported Go implementations — `gitutil.ValidateGitRef`, `gitutil.ValidateGitPath`, and `workflow.HasBashExplicitRestriction` — but no tests formally mapping spec predicates to their acceptance/rejection boundaries. Without predicate-level coverage, implementation drift from the formal spec (e.g., missing a NUL-byte or wildcard edge case) would be invisible to CI and could silently weaken the security guarantees the spec provides.

### Decision

We will add formal predicate-level unit tests for CTR-022 and CTR-023 in `pkg/gitutil/gitutil_ctr_formal_test.go` and `pkg/workflow/agent_validation_formal_test.go`. Each test function maps to a named formal predicate (P1–P12) derived from the spec's Z3/F* notation, covering acceptance boundaries (safe refs, safe paths, wildcard configs), rejection boundaries (hyphen-prefix, NUL bytes, traversal, absolute paths, empty values, false/empty-list/named-list restrictions), and the mixed-wildcard-among-names edge case identified in the implementation. This strategy makes the formal invariants executable and CI-enforced.

### Alternatives Considered

#### Alternative 1: Integration / End-to-End Tests Only

Test the CTR-022 and CTR-023 detectors through the full workflow scanning pipeline rather than at the function boundary. Integration tests already exist; the question is whether they are sufficient on their own.

Not chosen because integration tests do not isolate individual formal predicates — a regression in one boundary (e.g., NUL-byte rejection) might not trigger a failing integration scenario while still violating the spec. Mapping a CI failure back to a specific formal predicate also becomes much harder without unit-level coverage.

#### Alternative 2: Property-Based / Fuzz Testing

Use Go's native fuzzing (`go test -fuzz`) or a property-based library (e.g., `pgregory.net/rapid`) to generatively validate the validator contracts rather than enumerating cases.

Not chosen for this PR because the spec defines a finite set of well-understood predicates that can be exhaustively covered with table-driven unit tests. Fuzz testing adds tooling complexity and non-deterministic CI runtime without providing coverage that the explicit predicate tests miss. It would be a useful complement in the future, not a replacement.

### Consequences

#### Positive
- Formal spec predicates (P1–P12) become executable CI gates: any future change that breaks a boundary condition produces a named failing test with a direct link back to the predicate.
- Tests serve as living documentation of the CTR-022/CTR-023 acceptance/rejection semantics, reducing ambiguity when the spec or implementation needs to be updated.
- The mixed-wildcard-among-names edge case (P8 + implementation behavior) is now explicitly covered, closing a latent gap.

#### Negative
- The tests are tightly coupled to the current formal spec version (v1.0.20 predicates); if the spec is revised with new predicates or changed invariants, these tests will require corresponding updates, adding maintenance overhead.
- Tests live inside the implementation packages (`package gitutil`, `package workflow`), giving them access to unexported symbols but making them less useful as public API contracts.

#### Neutral
- Test style (table-driven, `testify/assert` + `testify/require`) is consistent with existing test conventions in the repository.
- The `//go:build !integration` tag keeps these tests out of the integration test suite and in the default unit-test pass, which is appropriate for pure predicate coverage.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
