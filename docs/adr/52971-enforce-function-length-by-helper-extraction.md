# ADR-52971: Enforce Function-Length Compliance via Helper Extraction

**Date**: 2026-08-15
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The repository's `golint-custom` linter enforces a 60-line maximum on function bodies across `pkg/workflow` and `pkg/cli`. As of this PR, 682 violations exist in this backlog. Long functions in `pkg/workflow/dependabot.go` — several exceeding 65–104 lines — were flagged. Letting the backlog grow unchecked makes adoption of the lint rule harder over time and reduces code readability and unit-testability of individual operations.

### Decision

We will reduce function-length violations by systematically extracting logical sub-operations from oversized functions into focused, named helper functions. We will not raise or suppress the lint limit. This PR addresses the 6 violations in `pkg/workflow/dependabot.go` as a representative slice of the broader 682-item backlog, reducing all 6 to under 60 lines while preserving public APIs and behavior exactly.

### Alternatives Considered

#### Alternative 1: Raise the function-length limit in golint-custom

The 60-line limit could be increased (or made per-file) to eliminate violations without code changes. This avoids the refactoring effort but retrenches the quality goal the lint rule was introduced to enforce, and makes functions harder to read and test in isolation.

#### Alternative 2: Add per-function nolint directives

Adding `//nolint:function-length` comments to each oversized function would silence the linter without changing the code. This preserves existing behavior but perpetuates oversized functions, provides no readability or testability benefit, and makes the lint rule meaningless for the files that need it most.

### Consequences

#### Positive
- All 6 targeted functions are now under the 60-line limit; `make golint-custom` reports 0 findings for `dependabot.go`.
- Extracted helpers (`loadOrInitPackageJSON`, `mergeExistingRequirements`, `loadOrInitGoModLines`, `appendGoModRequireSection`, `handleManifestGenerationError`, `reconcileGithubActionsIgnoreEntry`) are independently testable.
- Establishes a repeatable extraction pattern for the remaining 676 violations in the backlog.

#### Negative
- The package's unexported function count increases, widening the surface a reader must navigate.
- Callers must trace through an additional call level to follow end-to-end execution paths.
- Each refactoring PR must be reviewed carefully to preserve exact behavioral semantics (as noted in the PR for the `reconcileGithubActionsIgnoreEntry` unreachable-branch edge case).

#### Neutral
- Public APIs and behavior are unchanged; this is a pure internal refactoring.
- The broader 676-violation backlog is tracked separately (issue #52404); this PR is one planned installment.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
