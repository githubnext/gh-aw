# ADR-47252: Testify Error-Assertion Conventions — ErrorContains and require.*

**Date**: 2026-07-22
**Status**: Draft
**Deciders**: pelikhan (PR author), go-fan module review recommendations

---

### Context

The Go test suite used `assert.Contains(t, err.Error(), "…")` as a widespread pattern for checking error message substrings. This pattern has a latent defect: if `err` is `nil`, calling `.Error()` on it panics and crashes the entire test binary rather than recording a clean test failure. Separately, several error assertions used `assert.*` even when preceded by a `require.Error` precondition — meaning a failure in the preceding check would still allow subsequent assertions to execute, leading to confusing secondary failures. `.golangci.yml` already enables `testifylint` with `enable-all: true`, which flags both of these patterns.

### Decision

We will standardise on two rules for error assertions in Go tests:

1. **Use `ErrorContains(t, err, "…")` instead of `assert.Contains(t, err.Error(), "…")`** — `ErrorContains` validates that `err != nil` before inspecting the message, eliminating the nil-deref crash risk.
2. **Use `require.*` error assertions (not `assert.*`) after a `require.Error` precondition, and for any assertion whose failure would invalidate immediately following code** — `require` stops the test on first failure rather than allowing cascading false positives.

This decision is enforced mechanically across all `*_test.go` files; no production code was changed.

### Alternatives Considered

#### Alternative 1: Guard each call site with an explicit nil check

Add `if err != nil { assert.Contains(t, err.Error(), "…") }` before every affected assertion. This eliminates the panic without changing the assertion library API but is highly verbose (600+ sites), increases maintenance burden, and still does not address the `assert` vs `require` semantics issue.

#### Alternative 2: Keep `assert.Contains(t, err.Error(), "…")` and rely on test author discipline

Accept the current pattern and document that callers must ensure `err != nil` before calling `.Error()`. This has zero migration cost but preserves the crash risk and contradicts the existing `testifylint enable-all: true` policy, which was already configured to flag these patterns.

#### Alternative 3: Wrap error assertions in a custom helper function

Introduce a project-local `assertErrorContains(t, err, substr)` helper that handles the nil check internally. This is technically valid but duplicates functionality already provided by `testify` (`ErrorContains` was added in testify v1.7.1), adds an unnecessary abstraction, and reduces newcomer familiarity with the standard library.

### Consequences

#### Positive
- Nil-deref crash risk in error assertion paths is eliminated; a nil `err` now produces a clean `FAIL` line rather than a test binary crash.
- Test failures are reported at the first meaningful assertion (`require` stops execution), removing cascading false positives that obscure root causes.
- The codebase is now fully compliant with `testifylint enable-all: true`; no suppression flags or `//nolint` directives are needed.
- The pattern is idiomatic and familiar to any Go developer using testify — no custom DSL to learn.

#### Negative
- This is a 600+ file, 624-line mechanical change. Even though it is purely additive/substitutive in `*_test.go` files, the large diff creates merge conflicts for any concurrent branches that touch the same test files.
- Once merged, any contributor who reverts to the old `assert.Contains(t, err.Error(), "…")` pattern will see a CI lint failure — the stricter convention is now enforced by CI, not just recommended.

#### Neutral
- No change to test behaviour for passing tests; only the failure mode changes.
- No CI configuration changes required (`testifylint enable-all: true` was already present).
- Five files had their `assert` import removed as a side effect of the migration; unrelated `assert.*` calls in those files were also migrated as part of the sweep.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
