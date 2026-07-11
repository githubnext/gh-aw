# ADR-44888: Guard tolowerequalfold Linter Against Case-Incompatible Rewrites

**Date**: 2026-07-11
**Status**: Accepted
**Deciders**: gh-aw maintainers

---

### Context

The `tolowerequalfold` custom linter flags comparisons of the form `strings.ToLower(x) == y` and rewrites them to `strings.EqualFold(x, y)`. The original trigger condition was purely structural: it fired whenever one operand was a `strings.ToLower` or `strings.ToUpper` call (or a local variable aliasing such a call) and the other was any expression. This caused two classes of incorrect rewrites:

1. **Case-mismatched literals**: `strings.ToLower(name) == "ALICE"` is always false (the output of `ToLower` can never equal an uppercase string), so it is dead code — not a case-insensitive equality check. Rewriting it to `strings.EqualFold(name, "ALICE")` silently converts dead code into a live check.
2. **Mixed conversion functions**: `strings.ToLower(a) == strings.ToUpper(b)` is always false for any string containing letters, for the same reason. Rewriting to `EqualFold` introduces a behavioral change.

Both rewrites violate the invariant that an autofix must be behavior-preserving.

### Decision

We will make the `tolowerequalfold` rewrite **fail closed** and only permit cases we can prove equivalent. A comparison is flagged for EqualFold rewriting only when:

- one side is a `strings.ToLower`/`strings.ToUpper` call (or tracked alias), and
- the other side is an **ASCII string literal** that is already in the matching case (`ToLower` ↔ lowercase literal, `ToUpper` ↔ uppercase literal).

All other operand shapes are rejected (unknown variables, other function calls, and conversion-vs-conversion comparisons, including same-function pairs). This conservative choice avoids Unicode simple-fold edge cases (for example Greek sigma forms) where `ToLower`/`ToUpper` equality can diverge from `strings.EqualFold`.

The internal alias map type is changed from `map[types.Object]ast.Expr` to `map[types.Object]caseConvAliasInfo` to carry the function name alongside the argument, enabling alias-level guards.

### Alternatives Considered

#### Alternative 1: Suppress only mixed-case literals

Guard only against literals that contain both upper and lowercase characters (e.g., "Alice"). All-caps or all-lowercase literals paired with the wrong conversion function would still trigger the diagnostic.

Rejected because it does not cover the primary bug: `strings.ToLower(name) == "ALICE"` uses an all-uppercase literal and would still be incorrectly rewritten. The guard must be based on whether the literal's case is invariant under the conversion, not on whether it is "mixed case."

#### Alternative 2: Allow same-function conversion pairs (`ToLower(a)==ToLower(b)`)

Treat matching conversion functions as sufficient evidence of equivalence and continue rewriting those comparisons.

Rejected because Go Unicode semantics make this unsafe in general (for example, `strings.ToLower("ς")` is `"ς"` and `strings.ToLower("σ")` is `"σ"`, while `strings.EqualFold("ς", "σ")` is `true`). This would still permit behavior-changing rewrites.

### Consequences

#### Positive
- The linter no longer silently converts always-false (dead-code) comparisons into live case-insensitive checks.
- Mixed `ToLower`/`ToUpper` and same-function conversion-pair comparisons are excluded, preserving behavior across Unicode edge cases.
- Negative test fixtures lock in the new behavior and prevent regression.

#### Negative
- The rule becomes stricter and emits fewer diagnostics than before, intentionally favoring correctness over aggressiveness.
- The trigger condition is now more complex: `isEquivalentToEqualFold` delegates to compatibility helpers (`caseConvIsCompatible`, `caseConvAliasIsCompatible`, `literalCaseMatchesConv`, `stringLitValue`, `caseConvFuncAndArg`).
- The `caseConvAliasInfo` struct change is a breaking internal refactor: all functions accepting `map[types.Object]ast.Expr` must be updated to `map[types.Object]caseConvAliasInfo`.

#### Neutral
- The helper `caseConvFuncAndArg` is introduced as a single source of truth for extracting both the function name and argument from a conversion call; the existing `caseConvArg` and new `caseConvFuncName` become thin delegates to it.
- The literal guard uses ASCII-only matching so the analyzer does not need full Unicode fold-class reasoning.

---

*Finalized for PR #44888.*
