# ADR-44888: Guard tolowerequalfold Linter Against Case-Incompatible Rewrites

**Date**: 2026-07-11
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `tolowerequalfold` custom linter flags comparisons of the form `strings.ToLower(x) == y` and rewrites them to `strings.EqualFold(x, y)`. The original trigger condition was purely structural: it fired whenever one operand was a `strings.ToLower` or `strings.ToUpper` call (or a local variable aliasing such a call) and the other was any expression. This caused two classes of incorrect rewrites:

1. **Case-mismatched literals**: `strings.ToLower(name) == "ALICE"` is always false (the output of `ToLower` can never equal an uppercase string), so it is dead code — not a case-insensitive equality check. Rewriting it to `strings.EqualFold(name, "ALICE")` silently converts dead code into a live check.
2. **Mixed conversion functions**: `strings.ToLower(a) == strings.ToUpper(b)` is always false for any string containing letters, for the same reason. Rewriting to `EqualFold` introduces a behavioral change.

Both rewrites violate the invariant that an autofix must be behavior-preserving.

### Decision

We will add semantic compatibility guards to the linter's trigger condition. A comparison is flagged for EqualFold rewriting only when the operands are genuinely case-equivalent:

- If the non-conversion operand is a string literal, the literal must already be in the correct case for the conversion function (all-lowercase for `ToLower`, all-uppercase for `ToUpper`).
- If both operands are case-conversion calls, they must use the same function (`ToLower`/`ToLower` or `ToUpper`/`ToUpper`).
- These guards apply to both direct calls and local-variable aliases.

The internal alias map type is changed from `map[types.Object]ast.Expr` to `map[types.Object]caseConvAliasInfo` to carry the function name alongside the argument, enabling alias-level guards.

### Alternatives Considered

#### Alternative 1: Suppress only mixed-case literals

Guard only against literals that contain both upper and lowercase characters (e.g., "Alice"). All-caps or all-lowercase literals paired with the wrong conversion function would still trigger the diagnostic.

Rejected because it does not cover the primary bug: `strings.ToLower(name) == "ALICE"` uses an all-uppercase literal and would still be incorrectly rewritten. The guard must be based on whether the literal's case is invariant under the conversion, not on whether it is "mixed case."

#### Alternative 2: Require explicit nolint suppression for edge cases

Keep the current permissive trigger and require users to annotate problematic patterns with a `//nolint:tolowerequalfold` directive when they know the comparison is dead code.

Rejected because it puts the burden of correctness on every consumer of the linter rather than making the linter correct by default. A linter whose autofix can silently introduce bugs is more dangerous than no linter at all, especially in automated fix workflows.

### Consequences

#### Positive
- The linter no longer silently converts always-false (dead-code) comparisons into live case-insensitive checks.
- Mixed `ToLower`/`ToUpper` comparisons are correctly excluded from the diagnostic, preserving existing behavior.
- Negative test fixtures lock in the new behavior and prevent regression.

#### Negative
- The trigger condition is now more complex: the new `isEquivalentToEqualFold` function delegates to several helper functions (`caseConvIsCompatible`, `caseConvAliasIsCompatible`, `literalCaseMatchesConv`, `stringLitValue`, `caseConvFuncAndArg`), adding ~90 lines to the linter.
- The `caseConvAliasInfo` struct change is a breaking internal refactor: all functions accepting `map[types.Object]ast.Expr` must be updated to `map[types.Object]caseConvAliasInfo`.

#### Neutral
- The helper `caseConvFuncAndArg` is introduced as a single source of truth for extracting both the function name and argument from a conversion call; the existing `caseConvArg` and new `caseConvFuncName` become thin delegates to it.
- The guard uses Go's own `strings.ToLower`/`strings.ToUpper` at analysis time to determine whether a literal is in the correct case, ensuring the check is always consistent with the runtime behavior being linted.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
