# ADR-42834: Table-Driven Subtests for actionpins Pin-Resolution Coverage

**Date**: 2026-07-02
**Status**: Draft
**Deciders**: pelikhan (PR author)

---

### Context

The `pkg/actionpins` package implements critical GitHub Actions pin-resolution logic: semver fallback selection, exact-match hardcoded-pin resolution, and per-call warning deduplication via `PinContext.Warnings`. Prior to this PR, several internal branches (`findCompatiblePin`, `resolveExactHardcodedPin`, `resolveNonStrictHardcodedPin`) had no test coverage, leaving regression risk invisible. Test functions that did exist used flat top-level assertions with no subtest grouping, making failure attribution coarse-grained. The package also emits user-visible warnings to stderr as a side effect of resolution; that behavior was entirely untested.

### Decision

We will convert actionpins internal tests to table-driven subtests (`t.Run`) for parameterized cases and introduce `testutil.CaptureStderr` for asserting stderr side-effects of resolution functions. New dedicated test functions will cover the previously untested branches: semver major-version fallback, exact-version and exact-SHA resolution, no-match paths, and warning-dedup semantics across repeated calls.

### Alternatives Considered

#### Alternative 1: Flat top-level test functions (status quo)

Continue adding one `TestX` function per scenario without subtests. Simple to write and requires no `testutil` helper. Rejected because failure output names only the top-level function, making it harder to identify which of many inputs triggered the failure; it also encourages copy-paste duplication across similar cases.

#### Alternative 2: Interface-based stderr mock / io.Writer injection

Inject an `io.Writer` into the resolution functions to capture output in tests, rather than adding a `testutil.CaptureStderr` helper that redirects `os.Stderr`. Considered because it avoids OS-level file descriptor manipulation in tests. Rejected because it would require changing the signatures of internal (unexported) functions or adding an exported `Writer` field to `PinContext`, widening the public surface of the package solely for test concerns. The `testutil.CaptureStderr` approach keeps production code unchanged.

### Consequences

#### Positive
- Subtests provide precise failure attribution (the failing subtest name identifies the exact input, e.g., `TestFindCompatiblePin_SemverFallback/exact-major`) with no extra tooling.
- Previously untested resolution branches (semver fallback, exact-match, no-match, warning dedup) are now covered, reducing regression risk for critical pin-resolution logic.
- Table-driven structure consolidates related cases (e.g., two separate `TestGetContainerPin_MCPGatewayV*` functions merged into one) and makes adding future cases low-friction.

#### Negative
- Test file grows significantly (+170 lines), increasing the maintenance surface.
- `testutil.CaptureStderr` introduces a shared test-helper dependency; future changes to that utility can affect these tests indirectly.

#### Neutral
- No production code (`actionpins.go`) is modified; the decision affects only the test file and the `testutil` package import.
- Adoption of `assert.Zero` over `assert.Equal(t, 0, ...)` is a stylistic cleanup bundled into this PR; no behavioral change.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
