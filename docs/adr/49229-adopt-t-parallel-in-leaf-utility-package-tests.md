# ADR-49229: Adopt t.Parallel() in Leaf Utility Package Tests

**Date**: 2026-07-31
**Status**: Draft
**Deciders**: pelikhan, app/copilot-swe-agent

---

### Context

Only ~2.7% of test functions in the repository call `t.Parallel()`, leaving the `-parallel=4` concurrency budget in CI largely idle and CI wall-clock time higher than necessary. Four dependency-free leaf packages (`pkg/stringutil`, `pkg/fileutil`, `pkg/timeutil`, `pkg/constants`) contain pure read-only tests that are safe to parallelize — they do not mutate global state and do not share mutable fixtures between test cases. Go 1.22+ per-iteration loop variable semantics eliminate the classic closure-capture race that previously made parallelizing table-driven subtests risky.

### Decision

We will add `t.Parallel()` to all safe top-level test functions and their `t.Run` subtests across the four leaf packages, and explicitly exclude tests that mutate process-global state (e.g., those using `os.Setenv` or `t.Setenv`) because calling `t.Parallel()` in such tests causes a runtime panic. The change is applied in batch and verified with the `-race` flag.

### Alternatives Considered

#### Alternative 1: Keep Tests Sequential (Status Quo)

Continue not calling `t.Parallel()` in these packages. The approach has zero risk of global-state conflicts, but wastes available CI concurrency and leaves the parallelism budget consistently underutilized. Rejected because the benefit-to-risk ratio is poor: all four packages are read-only and safe to parallelize.

#### Alternative 2: Enforce Parallelism via the `paralleltest` Linter

Add the `paralleltest` linter to CI so every new test function is automatically flagged if it omits `t.Parallel()`. This would enforce the pattern going forward and flag the existing violations as a batch. Considered but deferred: linter adoption is a separate process-change that affects all packages, not just the four leaf packages targeted here. The manual batch approach achieves the immediate throughput goal without requiring project-wide linter policy changes.

### Consequences

#### Positive
- CI wall-clock time decreases as the `-parallel=4` concurrency budget is actually consumed by these packages.
- Tests concurrently exercising read-only constants and utility functions implicitly verify that they are safe to call from multiple goroutines.
- Establishes a visible example of the `t.Parallel()` pattern for contributors adding new tests to these packages.

#### Negative
- Three tests in `pkg/constants` and one test function in `pkg/fileutil` must be explicitly excluded because they call `os.Setenv` or `t.Setenv`; this exclusion list must be kept in sync as new state-mutating tests are added.
- Without the `paralleltest` linter (Alternative 2), new test functions added to these packages will silently omit `t.Parallel()` unless reviewers catch it manually — the pattern is not self-enforcing.

#### Neutral
- No production code is changed; the diff is entirely confined to `*_test.go` files.
- The change is targeted at four leaf packages only; other packages with global-state dependencies are unaffected by this PR and require separate analysis before being parallelized.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
