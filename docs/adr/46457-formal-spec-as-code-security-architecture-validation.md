# ADR-46457: Adopt Spec-as-Code for Security Architecture Validation

**Date**: 2026-07-18
**Status**: Draft
**Deciders**: Unknown

---

### Context

`specs/security-architecture-spec-validation.md` existed as a static cross-reference document auditing the 7-layer security architecture against compiled `.lock.yml` workflow files and JavaScript implementation. The document verified 15 normative predicates (job topology, input sanitization, permission isolation, threat-detection ordering, action pinning, timestamp validation, concurrency constraints), but those predicates were prose-only. Without automated verification, the spec could silently drift from the actual compiler and validator behaviour as the codebase evolved, creating a gap where the spec claimed conformance that was no longer true.

The project already has an established pattern of formal Go tests in `pkg/workflow` that drive the compiler and validator directly, so the infrastructure for this approach was already available.

### Decision

We will replace the static prose validation with 15 predicate-mapped Go tests (`TestFormalVAL_P1` through `TestFormalVAL_P15`) in `pkg/workflow/security_architecture_validation_formal_test.go`. Each test directly calls the production compiler or validator, compares compiled YAML output against expected security invariants, and is tagged `//go:build !integration` so it runs in the default unit-test suite. The formal model notation (TLA+, F*, Z3) is preserved in `specs/security-architecture-spec-validation.md` as the specification layer, while the Go tests serve as its executable binding.

### Alternatives Considered

#### Alternative 1: Keep Static Cross-References

Maintain the existing prose document with periodic manual re-validation. This is the lowest-friction approach, requires no new code, and keeps the spec human-readable without any coupling to internal compiler APIs. However, manual re-validation is error-prone and easy to skip; any refactor that changes compiler output or validator behaviour could silently invalidate the spec with no automated signal. Given the security-critical nature of the invariants (permission isolation, fork protection, threat-detection ordering), undetected drift is unacceptable.

#### Alternative 2: Run External Formal Verification Tools in CI (TLA+ / Z3 / F*)

Model-check the TLA+ and Z3 invariants directly using the TLA+ Toolbox or the Z3 solver as CI steps, and type-check the F* contracts. This provides stronger formal guarantees than Go tests (exhaustive model checking vs. example-based testing). However, it requires specialist tooling not currently present in the repository, introduces significant CI infrastructure complexity, and cannot exercise the actual Go compiler and validator code paths — so a bug in the compiler would still go undetected. The spec-as-code approach exercises real production paths and is immediately executable with the existing `go test` infrastructure.

### Consequences

#### Positive
- All 15 security predicates are verified on every CI run; spec drift is caught automatically before it reaches production.
- Tests call production compiler and validator code directly, so they detect behavioural regressions in the actual enforcement logic, not just in a separate model.
- The formal model notation is preserved in the spec document, giving both human-readable specification and machine-verifiable bindings in one place.

#### Negative
- The tests are tightly coupled to internal APIs (`compileFormalVALWorkflow`, `extractJobSection`, `sanitizeRunStepExpressions`, `validateConcurrencyQueueConfiguration`); any refactor of those functions requires updating the test suite.
- The `Generated Test Suite` section of the spec document contains an absolute filesystem path (`/home/runner/work/gh-aw/gh-aw/...`) that is environment-specific and will be incorrect outside the CI sandbox.
- Adding 15 additional tests increases unit-test suite runtime, though the impact is small since each test compiles a minimal synthetic workflow.

#### Neutral
- The formal notation sections (TLA+, F*, Z3/SMT-LIB) in the spec remain illustrative rather than directly executed; they serve as design documentation alongside the Go tests.
- The PR also adjusts the cron schedule for `agentic-auto-upgrade.yml` as an unrelated change bundled in the same commit.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
