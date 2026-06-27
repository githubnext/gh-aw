# ADR-41800: Enforce 60-Line Function Length Limit via Structural Extraction

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

A custom `largefunc` linter identified 660 function-length violations across the codebase. Five extreme hotspots in `pkg/workflow` and `pkg/cli` — with individual functions ranging from 145 to 650 lines — accounted for the most severe violations. Functions of this length are difficult to read, reason about, and test in isolation. The project enforces a 60-line function length limit as a quality gate, and these hotspots exceeded it by an order of magnitude.

### Decision

We will decompose oversized functions into smaller, focused private helpers that each stay under the 60-line limit, keeping all helpers within the same package as the original function. Decomposition is strictly structural: no behavioral changes are introduced, no new abstractions are added, and no new packages or interfaces are created. The refactoring affects `buildConclusionJob`, `buildPreActivationJob`, `extractPreActivationCustomFields`, `GetExecutionSteps`, `addWorkflowWithTracking`, `NewAddCommand`, `AuditWorkflowRun`, `NewAuditCommand`, and `auditJobRun`.

### Alternatives Considered

#### Alternative 1: Suppress or Raise the Linter Threshold

Disable the `largefunc` linter for these files or increase the line limit to accommodate the existing code. This would eliminate the lint failures without changing any code. We rejected this because it treats the symptom rather than the cause; the large functions represent genuine maintainability debt, and relaxing the threshold would allow the problem to grow unchecked over time.

#### Alternative 2: Reorganize Into Sub-packages

Move related logic groups from each large function into separate sub-packages (e.g., `pkg/workflow/conclusion`, `pkg/workflow/preactivation`). This would enforce clear boundaries through Go's package system rather than through naming conventions. We did not pursue this approach because it introduces a new package hierarchy that changes import paths and could break other callers, making it a larger, riskier change than a pure structural refactor within the existing packages.

### Consequences

#### Positive
- Each extracted helper is independently readable and fits within the 60-line cognitive budget
- Linter compliance is restored, unblocking CI and preventing future violations from being masked
- Individual helper functions are easier to unit-test in isolation
- Call sites in the original function become a readable sequence of named operations, serving as in-code documentation of the overall flow

#### Negative
- Execution traces now span more function frames, increasing indirection when stepping through a debugger
- The extracted helpers are unexported and tightly coupled to their parent function's call sequence; misuse or reordering by future contributors could introduce subtle bugs
- A larger number of small functions increases the cognitive overhead of navigating the file when reading from top to bottom

#### Neutral
- New helper files (`notify_comment_conclusion_helpers.go`) are added alongside existing files, slightly increasing the file count in affected packages
- `maps.Copy` replaces a manual map merge loop in one location, adopting a standard library idiom without changing semantics

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
