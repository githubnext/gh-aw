# ADR-49066: Extend stringsconcatloop to Detect `x = x + y` Accumulator Pattern

**Date**: 2026-07-30
**Status**: Draft
**Deciders**: pelikhan (copilot-swe-agent)

---

### Context

The `stringsconcatloop` linter existed to flag string `+=` inside for/range loops, which allocates a new string every iteration and creates O(n²) total-bytes cost for cross-iteration accumulators. However, the matcher only checked `token.ADD_ASSIGN` (`x += y`), leaving the semantically identical `x = x + y` (`token.ASSIGN` + `BinaryExpr{Op: ADD}`) undetected.

This gap was being deliberately exploited: PR #49033 explicitly rewrote 17+ `x += y` sites to `x = x + y` to bypass the linter, and the PR description stated this as its intent. A production site (`pkg/workflow/schedule_preprocessing.go:447`) already exhibited the exact pattern. If left unaddressed, `x = x + y` would become a documented, permanent workaround that other contributors could copy.

A naive "flag all `lhs = lhs + rhs` in loops" fix would cause false positives: when `lhs` is the loop's own iteration variable (e.g., `for _, line := range lines { line = line + suffix }`), the variable is reset each iteration and is not a cross-iteration accumulator — no O(n²) risk exists.

### Decision

We will extend `stringsconcatloop` to also match `token.ASSIGN` statements where `Lhs[0]` is an identifier whose name appears as the left operand of a `BinaryExpr{Op: ADD}` on the right-hand side (direct self-referential form only: `x = x + rhs`). A loop-scope guard (`isLoopScopedIdent`) will be added to exclude variables that are declared by the enclosing loop itself — range `Key`/`Value` identifiers and `ForStmt` `:=` init variables — so only genuine cross-iteration accumulators are flagged. The existing `enclosingLoopPosition` helper is refactored to `enclosingLoop`, returning the `ast.Node` needed by the guard.

### Alternatives Considered

#### Alternative 1: Do Nothing (Accept `x = x + y` as a Bypass)

Accept that `x = x + y` is a valid pattern and leave the linter unchanged. Developers wishing to avoid the `strings.Builder` requirement could use this form.

Not chosen because the bypass was explicitly documented and being actively promoted: PR #49033 demonstrated it as a deliberate strategy. Accepting it would permanently widen the enforcement gap and legitimize O(n²) string accumulation patterns.

#### Alternative 2: Flag All `x = x + y` Inside Loops Without a Loop-Scope Guard

Implement the `token.ASSIGN` + `BinaryExpr` check without adding `isLoopScopedIdent`, relying on the existing `FuncLit`-boundary stop alone.

Not chosen because `pkg/workflow/schedule_preprocessing.go:447` demonstrates a real false positive: `for _, line := range lines { line = line + ... }` where `line` is the range iteration variable reset each pass. Without the guard, valid single-iteration rebinds would be flagged, producing noise and eroding linter trust.

### Consequences

#### Positive
- Both `x += y` and `x = x + y` forms of cross-iteration string accumulation are now detected; the documented bypass vector is closed.
- The loop-scope guard prevents false positives on per-iteration variables, preserving the linter's signal-to-noise ratio.
- Testdata explicitly covers the new true-positive forms (range loop, classic for loop) and all false-positive guard cases (range value var, range key var, func literal boundary, non-self-referential `accum = p + accum`, out-of-loop form).

#### Negative
- The linter implementation is more complex: `enclosingLoop` now returns three values instead of two, and a new `isLoopScopedIdent` helper must be maintained.
- The scope guard only handles the direct single-step form `x = x + rhs`; chained forms such as `x = x + a + b` remain undetected by design, creating a narrower but still extant bypass for multi-operand expressions.

#### Neutral
- The `enclosingLoopPosition` function is renamed to `enclosingLoop` and its signature changes; callers (currently only `run()`) must be updated.
- `doc.go` and `Analyzer.Doc` descriptions are updated to reflect both detection forms, keeping documentation in sync with implementation.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
