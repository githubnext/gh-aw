# ADR-50848: Recognize Deferred Named Recover Helpers in goroutinemissingrecover

**Date**: 2026-08-06
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `goroutinemissingrecover` linter enforces that goroutines launched with a function literal contain a top-level `defer`/`recover` guard to contain panics locally. Previously the linter only recognized inline function literals as valid guards (`defer func() { recover() }()`). Goroutines deferring a named helper that directly calls `recover()` — a common idiomatic Go pattern — were falsely flagged as unprotected even though the Go spec defines recover as effective when called directly by the deferred function, regardless of whether that function is a literal or a named function. This caused false positives and forced teams to abandon helper-based recovery patterns in favour of the inline form.

### Decision

We will extend `hasTopLevelRecoverDefer` to resolve deferred call targets that are not function literals to their declared bodies (via two new helpers: `indexFuncBodies` and `resolveFuncBody`), then run the existing `containsRecoverCall` check against those bodies. This applies only to functions and methods declared in the same package and statically resolvable at analysis time. Func-valued variables, cross-package targets, and `recover()` calls nested inside closures within a helper are still treated as insufficient guards, preserving conservative false-negative behavior.

### Alternatives Considered

#### Alternative 1: Keep Inline-Only Detection

Accept the false positives and document that only inline function literals are accepted as recovery guards. This keeps the linter implementation minimal with no change needed. It was rejected because it requires callers to use a less idiomatic form and produces noise on valid, spec-compliant code, reducing trust in the linter.

#### Alternative 2: Resolve Cross-Package Named Functions

Extend resolution to include named functions declared in other packages by loading their source ASTs. This would eliminate false positives for shared recovery helper packages (e.g., a `recoverutil` package). It was rejected because the `go/analysis` framework does not provide cross-package AST bodies in a single pass; implementing it would require a separate pass dependency, significantly increasing complexity and analysis overhead.

### Consequences

#### Positive
- Eliminates false positives for valid recovery patterns using named in-package helpers, aligning linter behavior with the Go spec.
- Teams can extract `recover()` logic into a shared helper function without suppressing the linter, improving code organisation without sacrificing safety enforcement.

#### Negative
- Adds a pre-pass (`indexFuncBodies`) over all files in the analysed package on every analysis run, introducing a small but non-zero constant overhead per package.
- Named recover helpers defined in a different package are still flagged, which may surprise users who maintain a shared recovery helper package — they must inline recovery or suppress diagnostics.

#### Neutral
- The resolution logic uses `Origin()` to key generic function instantiations back to their declarations, so the approach naturally handles generic helpers without special-casing.
- The fix is additive: existing goroutines using inline literals continue to be accepted with no behavior change.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
