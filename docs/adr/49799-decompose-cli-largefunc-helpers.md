# ADR-49799: Decompose Oversized CLI Functions into Focused Helper Functions

**Date**: 2026-08-02
**Status**: Accepted
**Deciders**: pelikhan, app/copilot-swe-agent

---

### Context

The codebase enforces a custom `largefunc` linter rule capping functions at 60 lines (`make golangci-lint`). Several CLI entry-point and flow-control functions in `pkg/cli/` exceeded this limit: `RunAddInteractive` (126 lines), `createWorkflowPRAndConfigureSecret` (171 lines), `computeFirewallDiff` (144 lines), `parseSquidAccessLog` (80 lines), `buildAction` (83 lines), and `selectAIEngineAndKey` (118 lines). These functions mixed multiple sequential phases — host auto-detection, preflight checks, PR merge orchestration, secret configuration, and domain diff classification — into single units, making them difficult to test in isolation and harder to reason about under review.

### Decision

We will decompose each oversized CLI function into focused, single-responsibility helper functions co-located in the same file and package. Each extracted helper covers exactly one logical phase (e.g., host detection, PR merge loop, domain diff entry construction) and is tested independently where behavior is non-trivial. The `mergeAction` type and its constants, previously function-local, are promoted to package scope to allow multiple helpers to reference them without re-declaration.

### Alternatives Considered

#### Alternative 1: Raise the `largefunc` lint threshold

Increase the per-function line limit (or add per-file lint suppressions) to accommodate the current oversized functions without refactoring. This avoids code churn but leaves the underlying mixed-responsibility problem in place, degrades long-term readability, and weakens the lint rule for the rest of the codebase.

#### Alternative 2: Extract logic into separate packages or interface types

Move the orchestration phases into new dedicated packages or types rather than inline helper functions in the same file. This would provide module-level separation of concerns but would introduce new packages, potential import complexity, and substantial structural churn for what is fundamentally a sequential orchestration flow with no reuse across packages.

### Consequences

#### Positive
- All affected functions pass the 60-line `largefunc` lint check, unblocking CI.
- Extracted helpers are individually testable; focused unit tests were added for `prioritizeEngineOption` and `buildMergeOptions`.
- Orchestrator functions (`RunAddInteractive`, `createWorkflowPRAndConfigureSecret`) now read as a sequence of high-level named steps, improving code review clarity.

#### Negative
- Some extracted helpers carry high parameter counts (e.g., `processSquidAccessLogLine` takes five parameters) because context is passed down rather than captured in a receiver or struct — trading function-length compliance for increased parameter coupling.
- The call graph is deeper; understanding the full flow now requires tracing through more helper function boundaries.

#### Neutral
- Command behavior and semantics are fully preserved; existing tests continue to pass unmodified.
- The `mergeAction` type and constants are now package-level rather than closure-local; this widens their scope but does not expose them outside the package.

---

*ADR created by [adr-writer agent]. Status: Accepted.*
