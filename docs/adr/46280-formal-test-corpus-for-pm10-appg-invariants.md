# ADR-46280: Encode PM-10 and Appendix G Security Invariants as Formal Go Tests in the Workflow Compiler Test Corpus

**Date**: 2026-07-18
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The gh-aw workflow compiler is governed by a W3C-style security architecture specification (`specs/security-architecture-spec-summary.md`) that formally defines security invariants using TLA+, F*, and Z3/SMT-LIB notation. Two invariant groups — PM-10 (pre-activation RBAC topology and permission constraints) and Appendix G (lock-file SHA-pinning and fork-filter validation) — had no corresponding automated checks in the compiler's test suite. This left a gap between the written spec and the compiled YAML output, meaning regressions in these invariants would not be caught by CI. The fix closes issue #46279 by adding a dedicated formal test file in `pkg/workflow/` that exercises the compiler's output directly.

### Decision

We will add explicit formal-style Go unit tests in `pkg/workflow/security_architecture_pm10_formal_test.go` that assert the PM-10 and Appendix G invariants against the live compiler output. Each test name mirrors the corresponding spec section (PM10a–PM10d, AppG1–AppG2), and tests compile a minimal workflow frontmatter through the full `NewCompiler → ParseWorkflowString → CompileToYAML` pipeline and then assert structural properties of the resulting YAML. This makes the spec-to-code traceability explicit and catches regressions automatically in CI.

### Alternatives Considered

#### Alternative 1: Property-Based / Fuzz Testing

Use Go's `testing/quick` package or a fuzzer (e.g., `go-fuzz`) to generate random workflow frontmatter inputs and assert that no random input can produce a compiled YAML that violates PM-10 or AppG invariants. This would provide broader coverage over the input space rather than relying on hand-crafted examples. It was not chosen because the PM-10 and AppG invariants are structural properties of specific trigger configurations (e.g., "when `roles` is non-empty, `pre_activation` must precede `activation`"), which are most clearly expressed as deterministic assertions over representative inputs. Fuzz testing would add significant tooling complexity without improving the precision of the targeted invariant checks.

#### Alternative 2: Integration Tests Against Live Workflow Runs

Validate PM-10 and AppG behaviors through end-to-end integration tests that trigger actual GitHub Actions workflow runs and observe their execution topology. This would test the runtime behavior rather than the compiled YAML. It was not chosen because the invariants are compiler output invariants — they govern what the compiler emits — not runtime behaviors. Integration tests are slow, require secrets, and are tagged with `//go:build integration`; the PM-10/AppG checks are fast deterministic assertions that belong in the non-integration unit test suite.

#### Alternative 3: Static Spec Linting (Doc-Level Verification)

Write a linter that parses the spec Markdown and checks cross-references between spec sections and existing test function names, flagging sections with no matching test. This would identify spec coverage gaps without adding new test logic. It was not chosen because the goal is to actually verify the compiled output conforms to the invariants, not merely to confirm that a test name exists. Spec linting cannot substitute for executing the compiler and inspecting the result.

### Consequences

#### Positive
- PM-10 (pre-activation RBAC topology, permission isolation, required-roles defaults) and Appendix G (SHA-pinning, fork-filter) invariants are now machine-checked on every CI run, preventing silent regressions.
- Test names like `TestFormalPM10a_PreActivationJobPrecedesActivation` and `TestFormalAppG1_CompiledStepsUseSHAPins` create explicit, searchable traceability from spec section to executable assertion.
- Table-driven variants in `TestFormalPM10c_PreActivationPermissionsTableDriven` ensure the read-only permission invariant holds across multiple frontmatter configurations without code duplication.

#### Negative
- Tests invoke the full compiler pipeline (`NewCompiler`, `ParseWorkflowString`, `CompileToYAML`), so internal compiler refactors that change YAML structure or API signatures will require corresponding test updates.
- The tests depend on `extractJobSection()`, a shared test helper whose contract is assumed stable; any change to that helper may silently break invariant assertions if not carefully reviewed.

#### Neutral
- The new file is tagged `//go:build !integration`, keeping it in the default `go test ./...` run and out of the slower integration suite.
- Lock-file manifests in `.github/workflows/*.lock.yml` receive minor comment-label cleanup (SHA-only labels updated to version tags) as a side effect of the same PR.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
