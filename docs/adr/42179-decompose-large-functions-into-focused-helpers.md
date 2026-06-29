# ADR-42179: Decompose Large Functions into Focused Helper Functions

**Date**: 2026-06-29
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The codebase's `largefunc` linter (LintMonster) flagged 10 functions across `cmd/gh-aw/main.go`, `pkg/console/console.go`, `pkg/github/label_objective_mapping.go`, `pkg/parser/schema_suggestions.go`, and four custom linter packages as exceeding the maximum allowed function length. Large functions reduce readability, hinder focused unit testing, and violate the single-responsibility principle. Because these functions mixed flag extraction, logic branching, rendering, and output formatting in a single body, changes to one concern required navigating unrelated code. Automated lint enforcement created a concrete, non-deferrable requirement to reduce function size.

### Decision

We will decompose each flagged large function into a set of focused, named helper functions co-located in the same source file, preserving all existing behavior without introducing new packages, interfaces, or exported symbols. Helper names follow the pattern `<topic><Concern>` (e.g., `tableTTYCheck`, `compileCommandOptionsFromFlags`) to make the call site self-documenting. The `largefunc` lint rule threshold is treated as a hard constraint rather than something to suppress or tune.

### Alternatives Considered

#### Alternative 1: Suppress or raise the `largefunc` threshold

The lint rule could be disabled per-file (`//nolint:largefunc`) or the global threshold could be raised to silence the violations without changing code. This was rejected because suppression hides growing complexity and makes future violations harder to notice; raising the threshold postpones rather than resolves the underlying readability problem and undermines the purpose of automated quality gates.

#### Alternative 2: Split large files into separate sub-packages or files

`cmd/gh-aw/main.go` could have been split into multiple files (e.g., `compile_cmd.go`, `help_cmd.go`), and `pkg/console/console.go` could have separated rendering logic into a `pkg/console/table/` sub-package. This would provide stronger encapsulation and independently testable units. It was not chosen because it carries a higher refactoring cost (package renaming, import cycle checks, re-exported types), risks broader breakage across dependents, and is disproportionate to the immediate goal of clearing the lint backlog. This alternative remains viable for a follow-on refactor.

### Consequences

#### Positive
- All 10 flagged functions now satisfy the `largefunc` lint rule, unblocking the quality-sweep batch.
- Individual helper functions (e.g., `computeObjectiveValueSum`, `tableBorderStyle`) are small enough to be independently unit-tested.
- Call sites in the refactored entry points (`init()`, `RenderTable`, `ComputeObjectiveValue`) read as structured, step-by-step prose rather than dense imperative blocks.
- No exported API surface or behavior changes — existing callers and tests are unaffected.

#### Negative
- Execution traces now span more stack frames, making it marginally harder to follow flow in a debugger or stack trace without an IDE.
- Helper functions are not independently exported or documented, so their discoverability outside the immediate file context depends on naming conventions being followed consistently.

#### Neutral
- The change is limited to internal restructuring within existing files/packages; no new dependencies, build-time costs, or configuration changes are introduced.
- The approach establishes an implicit convention (decompose-in-place rather than split-package) for how future lint violations should be addressed, which may need to be documented in contribution guidelines.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
