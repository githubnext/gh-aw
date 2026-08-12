# ADR-52325: White-Box Testing for Permission Management Spec-Compliance Predicates

**Date**: 2026-08-12
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`specs/security-architecture-spec-validation.md` §12 Compliance Test Matrix Gap Analysis flagged three Permission Management test cases — T-PM-003 (strict mode gating), T-PM-005 (`workflow_run` repository validation), and T-PM-007 (token precedence) — as lacking dedicated test coverage. The implementation functions that encode these behaviors (`hasWorkflowRunTrigger`, `buildWorkflowRunRepoSafetyCondition`, `validateWorkflowRunBranches`, `getEffectiveGitHubToken`, `getEffectiveSafeOutputGitHubToken`) are intentionally unexported internals within `pkg/workflow`. Closing the spec coverage gap requires a test file that can call these unexported functions directly. The same white-box pattern is already used by existing `security_architecture_*_formal_test.go` files in this package.

### Decision

We will declare the formal spec-compliance test file (`security_architecture_pm_formal_test.go`) as `package workflow` (same-package / white-box test) rather than `package workflow_test` (external / black-box test). This allows the test file to call unexported functions directly without requiring any changes to production code or the addition of exported shims.

### Alternatives Considered

#### Alternative 1: Add Exported `*ForTest()` Wrapper Functions

Thin exported shims (e.g., `HasWorkflowRunTriggerForTest`) are added in a separate `_test.go` file within `package workflow`, making the internals callable from `package workflow_test`. This keeps the external-test boundary intact but requires writing and maintaining boilerplate wrapper functions for each unexported symbol under test. Rejected because it adds production-adjacent indirection without a correctness benefit over same-package membership, and the project has already established the white-box convention for this file family.

#### Alternative 2: Promote the Functions to Exported Status

Rename the unexported functions to uppercase exports so they become part of the package's public API. Rejected because these are deliberate implementation details — exporting them would make them an implicit contract that is harder to change later, and the motivation for testing them is compliance verification, not external use.

### Consequences

#### Positive
- No changes to production code are needed; the implementation is tested exactly as written.
- Follows the existing project convention for `security_architecture_*_formal_test.go` files, keeping this new file consistent with its siblings.
- Predicate tests directly encode the spec invariants against the real functions, avoiding any mismatch between a shim's behavior and the underlying implementation.

#### Negative
- The test file is tightly coupled to internal function signatures; renaming or extracting methods to a different struct will break these tests without any compiler warning at the call site.
- White-box tests carry a risk of circular reasoning: they verify properties of the same code they are derived from, so they will not catch cases where the implementation and the tests share the same misunderstanding of the spec.

#### Neutral
- The test file must be declared in `package workflow` to compile, placing it in the same namespace as production code.
- The `//go:build !integration` build tag ensures the file is excluded from integration builds, consistent with other unit tests in the package.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
