# ADR-52295: Targeted Pure-Function Test Coverage via PureLock

**Date**: 2026-08-12
**Status**: Draft
**Deciders**: pelikhan, PureLock automation

---

### Context

The `pkg/workflow` package had several pure functions with 0% function coverage:
`parseSamplesValue`, `ErrorSeverity.Heading`, and `ErrorSeverity.Icon`. These functions
are fully deterministic (no I/O, no shared state, no side effects), which makes them
ideal candidates for exhaustive, table-driven unit tests. The team uses PureLock — an
automated CI workflow — to identify zero-coverage pure functions and generate targeted
test suites for them, submitting each sweep as a PR.

### Decision

We will use the PureLock workflow to automatically generate table-driven test suites for
zero-coverage pure functions and merge those tests into `pkg/workflow/` via dedicated
PRs. Each generated test file targets a single function or closely related group of
functions, uses `testify/assert` for assertions, and enumerates all meaningful input
shapes including edge cases (nil, empty, mixed types, unsupported scalars).

### Alternatives Considered

#### Alternative 1: Accept Coverage Gaps for Utility Functions

Treat pure helper functions as low-risk and exempt them from formal test coverage
requirements. Coverage gaps would persist until an unrelated bug or regression forced
attention. Rejected because zero-coverage functions cannot be safely refactored, and
silent regressions in type-switch logic (e.g., unexpected `nil` returns) are hard to
detect at integration level.

#### Alternative 2: Cover These Functions via Integration or Behavioral Tests

Exercise `parseSamplesValue`, `Heading`, and `Icon` indirectly through higher-level
workflow execution tests. Rejected because indirect coverage obscures which input
shapes are actually exercised, integration tests are slower and more fragile, and it
becomes harder to attribute regressions to these specific functions when a higher-level
test fails.

### Consequences

#### Positive
- Package function coverage advances from 87.05% to 87.10%, improving regression
  detection for the `workflow` package.
- Table-driven tests for pure functions are deterministic, require no fixtures or mocks,
  and run in milliseconds — they add no flakiness risk.
- PureLock PRs create an explicit record of which functions were deliberately covered
  and what edge cases were considered.

#### Negative
- Each PureLock sweep adds test-maintenance surface: if a covered function's signature
  or behavior changes, tests must be updated alongside it.
- Adding 203 lines of test code crosses the ADR-gate volume threshold, requiring
  architectural review for PRs that contain no production logic changes — this may feel
  heavyweight for test-only work.

#### Neutral
- The `testify/assert` library is already a project dependency; no new dependencies are
  introduced.
- PureLock runs as an automated bot PR (label: `automation`, `testing`, `coverage`),
  which is distinguishable from human-authored changes in the PR history.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
