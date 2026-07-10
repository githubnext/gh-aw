# ADR-44590: Adopt Formal Verification Methods for Security Architecture Invariants

**Date**: 2026-07-10
**Status**: Draft
**Deciders**: Unknown (AI agent — copilot-swe-agent; review and confirm before Accepted)

---

### Context

`specs/security-architecture-spec-summary.md` defines 7 security guarantees (SG-01 through SG-07) for the gh-aw workflow compiler's 7-layer security architecture. Prior to this change, these guarantees existed as natural-language requirements and RFC 2119 statements only — they had no executable, continuously-verified form. Without runnable invariants, a regression in any guarantee would not be caught automatically by CI, leaving security properties open to silent drift as the compiler evolves.

### Decision

We will adopt formal specification methods (TLA+ state-machine invariants, F* pre/post contracts, and Z3/SMT-LIB arithmetic bounds) to document each security guarantee, and we will encode each invariant as a Go unit test (`//go:build !integration`) that calls production code directly without stubs or mocks. The 15 test functions are mapped 1:1 to invariant predicates in a Behavioral Coverage Map embedded in the spec summary, creating bidirectional traceability between the formal model and the CI test suite.

### Alternatives Considered

#### Alternative 1: Property-based testing (gopter / rapid)

Generative/fuzzing libraries such as `gopter` or `rapid` can automatically discover edge cases by generating random inputs. They were considered because the security invariants (e.g., "no write permission in any scope") are naturally quantified over all inputs. They were not chosen because: (a) the existing invariants have small, enumerable state spaces that are exhaustively covered by hand-written tests; (b) adding a property-testing library is a non-trivial dependency with its own learning curve; and (c) the explicit assertion messages naming the SG identifier and invariant clause are essential for clear CI feedback, which property libraries make harder to produce.

#### Alternative 2: Keep formal model as standalone documentation artifacts (TLA+ spec files only)

A common approach is to maintain `.tla` / `.fst` / `.smt2` files as separate artifacts checked into the repo and verified by a dedicated model-checker step (e.g., the TLA+ toolbox or F* nightly). This gives stronger mathematical guarantees than Go tests. It was not chosen because: (a) the toolchain (TLC, F* nightly) is not currently part of the CI infrastructure and would require significant setup; (b) the production code is Go, not a formalized language — connecting the formal model to the actual implementation would still require a manual mapping layer; and (c) the Go test approach gives immediate CI value without new tooling.

#### Alternative 3: Integration tests over the full compiled workflow

Each invariant could be verified by standing up a complete workflow, compiling it end-to-end, and asserting properties of the emitted YAML. Integration tests were considered because they exercise the full stack and are less coupled to internal details. They were not chosen because: (a) the `//go:build integration` suite is already separate and slower; (b) the security guarantees being tested here are compiler-level invariants best verified at unit scope; and (c) calling internal functions (e.g., `sanitizeRunStepExpressions`, `validateDangerousPermissions`) directly gives tighter feedback loop when a specific invariant regresses.

### Consequences

#### Positive
- Each of the 7 security guarantees is now a continuously-verified CI invariant; regressions in any guarantee produce a named, actionable test failure.
- The Behavioral Coverage Map in the spec summary creates explicit traceability from formal predicate to test function, making security audits easier.
- Tests call production code without stubs, so they reflect the real compiler behavior rather than a mocked approximation.
- The formal model notation (TLA+/F*/Z3) in the spec provides a precision level that natural-language requirements cannot, reducing ambiguity during future spec reviews.

#### Negative
- The test file (`security_architecture_sg_formal_test.go`) accesses unexported functions in the `workflow` package; any internal refactoring of those functions requires corresponding test updates, creating maintenance coupling.
- The TLA+/F*/Z3 notation in `specs/security-architecture-spec-summary.md` is mathematical formalism that contributors without a formal methods background may find opaque; there is no tooling to verify the formal model itself (it is documentation-only).
- Adding 15 tests to the `!integration` suite increases unit-test runtime; tests that compile real workflows (SG-06, SG-07, BasicConformance, JobTopology) write temporary files via `os.WriteFile`, which adds filesystem I/O to what might otherwise be purely in-memory tests.

#### Neutral
- The 15 tests use the `testify` library (`assert`/`require`), which is already a project dependency — no new dependencies are introduced.
- The `formalJobSections` and `formalJobOrderValid` helpers referenced in `TestFormalJobTopology_PipelineOrderEnforced` are defined in a sibling test file (`security_architecture_formal_test.go`) in the same package and build tag; both files must be compiled together.
- The formal model in the spec is documentary, not executable — it is not verified by a model checker and serves as a human-readable specification anchor rather than a mechanically-checked proof.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
