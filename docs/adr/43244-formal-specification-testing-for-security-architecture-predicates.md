# ADR-43244: Formal Specification Testing for Security Architecture Predicates

**Date**: 2026-07-04
**Status**: Draft
**Deciders**: Unknown

---

### Context

The gh-aw 7-layer security architecture is defined in `specs/security-architecture-spec-summary.md` as 10 formal predicates (P1–P10), covering invariants such as input sanitization, agent permission boundaries, network allowlists, sandbox enforcement, and token isolation. Without executable tests tied to those predicates, compliance is checked only at code-review time and can silently regress when production helpers are refactored. The codebase already contains a `*_formal_test.go` convention in `pkg/workflow/` for encoding spec-level invariants as Go tests. This decision extends that pattern to cover all 10 security-architecture predicates and introduces file-local formal helpers for predicates that do not map to a single production call site.

### Decision

We will add `pkg/workflow/security_architecture_formal_test.go` containing 10 `TestFormal_P*` functions—one per predicate—that call production code directly with no stubs, plus four file-local formal helpers (`formalConformanceMonotonicity`, `formalJobOrderValid`, `formalTokenAbsentFromEnv`, `formalValidationBlocksEmit`) for predicates whose invariants span multiple call sites. This makes the formal spec continuously verifiable in CI under the default (non-integration) build tag.

### Alternatives Considered

#### Alternative 1: Manual Code-Review Enforcement

Each PR touching security-sensitive code would rely on reviewers to verify predicate compliance against the spec document. This is simple to introduce (no new files) and imposes no coupling to production function signatures. It was not chosen because review coverage is inconsistent, spec drift is invisible to CI, and regressions in production helpers are not caught until the next human review.

#### Alternative 2: Integration-Level Security Tests

Predicates could be verified at the system level (full workflow compilation + execution) rather than at the unit level. This would test the full call stack and avoid direct coupling to internal function signatures. It was not chosen because integration tests are slower and gated behind the `integration` build tag, which is not part of every CI run; unit-level tests provide faster feedback and run unconditionally.

#### Alternative 3: Embedding Invariant Checks in Production Code (Assertions/Panics)

Security invariants could be enforced at runtime in production code via `panic` or explicit assertion helpers, ensuring they are checked during normal execution. This was not chosen because it conflates validation logic with the production hot path, adds noise to error messages surfaced to users, and makes it harder to reason about which checks are spec-mandated versus operational.

### Consequences

#### Positive
- All 10 security predicates become continuously verifiable in CI without any special build flags, catching regressions automatically.
- The test file acts as a living, executable cross-reference between the formal spec and the production codebase—each predicate maps one-to-one to a named test.
- File-local formal helpers make spec-level invariants (e.g., monotonicity, emit-gate logic) explicit and independently readable, even when no single production function encapsulates them.

#### Negative
- Tests are tightly coupled to specific internal function signatures (`sanitizeRunStepExpressions`, `validateDangerousPermissions`, `isSandboxEnabled`, etc.); renaming or restructuring these functions breaks tests regardless of behavioral equivalence, increasing refactor friction.
- File-local formal helpers (`formalConformanceMonotonicity`, etc.) duplicate or paraphrase spec logic outside the canonical spec document; they may drift from `specs/security-architecture-spec-summary.md` as the spec evolves if not updated in tandem.

#### Neutral
- The `//go:build !integration` tag places these tests in the default unit-test suite, consistent with the existing `*_formal_test.go` pattern in the package.
- Each predicate is mapped to exactly one test function, creating a traceable spec-to-test matrix that can be audited against the spec appendix.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
