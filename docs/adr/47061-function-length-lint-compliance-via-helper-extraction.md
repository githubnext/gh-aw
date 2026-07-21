# ADR-47061: Function-Length Lint Compliance via Private Helper Extraction

**Date**: 2026-07-21
**Status**: Draft
**Deciders**: Unknown (automated refactoring by copilot-swe-agent)

---

### Context

`make golint-custom` reported 680 long-function findings (functions exceeding 60 lines) across `pkg/workflow` and `pkg/cli`. The worst offenders ranged from 200 to 961 lines (e.g., `buildMaintenanceWorkflowYAML` at 961 lines, `commentOutProcessedFieldsInOnSection` at 697 lines). The existing lint rule enforces a maintainability threshold; at this scale the violations blocked the codebase from passing lint checks cleanly and made individual functions difficult to read or test in isolation.

### Decision

We will eliminate function-length lint violations by extracting cohesive logical blocks into focused private helper functions and, where appropriate, splitting large source files into companion files (e.g., `maintenance_workflow_yaml_jobs.go`). Public interfaces and observable behavior are preserved unchanged. This approach removes all 680 findings without modifying the lint threshold or suppressing rules.

### Alternatives Considered

#### Alternative 1: Per-function `//nolint:funlen` suppression directives

Add inline suppression comments at each offending function to silence the lint rule without changing code structure. This is faster to apply but permanently exempts those functions from the length rule, masks future growth, and provides no readability or testability benefit.

#### Alternative 2: Raise or remove the function-length threshold in lint configuration

Increase the allowed line count (e.g., from 60 to 200) or disable the rule entirely across the package. This eliminates all current findings instantly but weakens the quality gate for the entire codebase going forward and does not address the underlying complexity in those functions.

#### Alternative 3: Struct-based decomposition using methods on new types

Introduce new intermediate types (structs or interfaces) and attach the extracted logic as methods. This provides stronger encapsulation and enables dependency injection for testing but requires deeper structural changes, risks altering the API surface, and is disproportionate effort for a mechanical lint-compliance refactor.

### Consequences

#### Positive
- All 680 function-length lint findings are eliminated; `make golint-custom` passes cleanly.
- Individual helper functions are shorter and independently testable.
- Companion files (`maintenance_workflow_yaml_jobs.go`) group related job-builder logic, improving discoverability.
- A pre-existing bug in `applyRepoMemoryDefaultBranch` (early-exit due to `BranchName` being pre-populated) is fixed as a side effect of the decomposition.

#### Negative
- Execution flow is now distributed across more call frames, increasing the cognitive load needed to trace end-to-end behavior.
- The number of private functions and, in one case, source files increases, adding navigation overhead.
- Helper functions are private and closely coupled to their single caller; they carry no reuse benefit today.

#### Neutral
- No changes to public function signatures, exported types, or test files are required.
- The lint threshold itself (60 lines) remains unchanged — this ADR addresses compliance rather than the threshold value.
- Future growth in the refactored functions will still be caught by the same lint rule.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
