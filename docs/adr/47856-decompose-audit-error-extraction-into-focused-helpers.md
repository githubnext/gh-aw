# ADR-47856: Decompose Audit Error Extraction into Focused Helpers

**Date**: 2026-07-24
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/cli/audit_report.go` contains `extractPreAgentStepErrors`, a function responsible for finding failure signals in GitHub Actions workflow logs. It was bundling four distinct concerns into a single body: scanning flat job log files, scanning nested step log files, extracting agent-stdio failure excerpts, and selecting the final fallback. The function exceeded the project's `largefunc` lint limit, making it a target in the shared lint-monster backlog (issue #47448). The decision is scoped to the `pkg/cli` audit-report path and does not affect other packages.

### Decision

We will decompose `extractPreAgentStepErrors` into single-purpose helper functions (`scanWorkflowStepLogs`, `scanFlatStepLog`, `scanNestedStepLogs`, `updateLastStep`, `appendErrorAnnotation`, `extractGHErrorLines`, `extractAgentFailureError`, `extractLastStepFallbackError`) and extract `classifyAgentStdioLines` from `extractAgentStdioFailureExcerpt`. The orchestrating function becomes a short call sequence that makes the fallback priority order (annotations → agent-stdio → last-step) explicit. No behavior change is introduced.

### Alternatives Considered

#### Alternative 1: Suppress the linter for this function

Add a `//nolint:cyclop` or similar inline directive to silence the `largefunc` rule for `extractPreAgentStepErrors`. This would satisfy the CI gate with zero code change but would leave the complex monolith in place, making it harder to read and test individual scanning paths. It also sets a precedent for suppression over improvement that conflicts with the project's lint-monster initiative.

#### Alternative 2: Introduce a scanning struct with methods

Define a `stepLogScanner` struct that holds iteration state (`lastStep`, `errorAnnotations`) and exposes methods (`scanFlat`, `scanNested`). This collocates mutable state with behavior and avoids passing accumulator slices through function arguments. It was not chosen because the scanning state does not persist beyond a single call to `extractPreAgentStepErrors`; a struct would add ceremony without a lifetime justification. The functional approach with explicit parameter passing is idiomatic Go for short-lived, stateless iteration helpers.

### Consequences

#### Positive
- Each helper function is small enough to unit-test in isolation (e.g., `scanFlatStepLog`, `extractGHErrorLines`).
- `extractPreAgentStepErrors` is now a readable orchestrator whose three-branch fallback priority is self-documenting.
- All affected functions fall within the `largefunc` limit, eliminating the lint violation for this slice of the backlog.

#### Negative
- The `stepLog` struct is promoted from a function-local type to package scope, giving it a wider visibility than strictly necessary.
- The package now has significantly more top-level function names, increasing the surface area that new contributors need to navigate.

#### Neutral
- The refactor is behavior-preserving; existing tests cover the entry points and continue to pass without modification.
- The `appendErrorAnnotation` function takes a `flat bool` parameter to select the log message format; this is a minor code smell but avoids duplicating the annotation-building logic entirely.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
