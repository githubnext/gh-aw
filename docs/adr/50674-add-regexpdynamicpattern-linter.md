# ADR-50674: Add regexpdynamicpattern Static Analysis Linter

**Date**: 2026-08-05
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `pkg/linters` package provides custom Go static analysis passes that enforce codebase-wide safety
and style invariants. `regexp.Compile` and `regexp.MustCompile` are safe when called with compile-time
constant patterns, but when the pattern is built from dynamic input (e.g. `fmt.Sprintf`, string
concatenation with variables, or function parameters) two distinct risks arise: a malformed pattern
causes a runtime panic in `MustCompile` variants (or a returned error that callers often ignore),
and if the dynamic portion is influenced by untrusted input the attacker can control pattern
complexity or size. The existing
`regexpcompileinfunction` linter only enforces *where* compilation occurs (package-level vs.
function-level), not *what* is being compiled, leaving the dynamic-pattern risk unaddressed.

### Decision

We will introduce a new `regexpdynamicpattern` analyzer in `pkg/linters/regexpdynamicpattern/` that
reports any `regexp.Compile`, `regexp.MustCompile`, `regexp.CompilePOSIX`, or
`regexp.MustCompilePOSIX` call whose first argument is not a compile-time constant string (literal,
`const` identifier, or constant-only expression). Package identity is
resolved via the type checker to handle aliased imports without false positives. The analyzer is
registered in `pkg/linters/registry.go` alongside the existing linters and respects
`//nolint:regexpdynamicpattern` suppressions for intentional dynamic patterns.

### Alternatives Considered

#### Alternative 1: Rely solely on the existing `regexpcompileinfunction` linter

`regexpcompileinfunction` enforces that regexp compilation occurs at package level. A package-level
`var re = regexp.MustCompile(buildPattern())` would still pass that linter while introducing a
dynamic-pattern risk. This alternative was rejected because it does not address the class of risk
identified: the *content* of the pattern, not its *location*, determines the safety hazard.

#### Alternative 2: Runtime validation wrapper

Wrap regexp compile calls with a project-internal helper that validates or sanitizes the pattern at
runtime. This would catch panics or errors but cannot prevent untrusted input from controlling
pattern complexity or size, and introduces runtime overhead on every compilation call. It also
requires migrating all call sites, whereas a static linter operates without any code changes to call
sites that already use constant patterns.

### Consequences

#### Positive
- Eliminates an entire class of runtime panics caused by malformed dynamically-constructed regexp patterns.
- Flags call sites where untrusted input could control regexp pattern complexity or size.
- Consistent with the project's existing philosophy of enforcing safety invariants at analysis time rather than at runtime.

#### Negative
- Intentional dynamic regexp patterns (e.g. test helpers that build patterns from parameters) require `//nolint:regexpdynamicpattern` suppressions, adding annotation noise.
- The linter operates only on packages compiled with full type-checker information; packages analyzed without `TypesInfo` populated will silently skip pattern-constant checks (the analyzer returns `false` for unknown patterns rather than reporting a finding).

#### Neutral
- The `pkg/linters/doc.go` active-analyzer count increments from 62 to 63; documentation and `spec_test.go` must be updated whenever the analyzer list changes (this is already the project convention).
- The new analyzer composes with the existing `nolint` and `filecheck` infrastructure (generated-file skipping, suppression directives) without requiring any changes to those internal packages.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
