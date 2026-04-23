# Pre-Consolidation Analysis: `pkg/workflow` Expression Subsystem

## Overview

The expression subsystem spans **9 production files** and **7+ test files** in `pkg/workflow/`.
The consolidation target is `copilot/consolidate-optimization-work`.

---

## File Inventory

### Production Files

| File | Lines | Responsibility |
|------|------:|----------------|
| `expression_nodes.go` | 196 | AST node types (`ConditionNode`, `AndNode`, `OrNode`, `NotNode`, `DisjunctionNode`, `FunctionCallNode`, `PropertyAccessNode`, `StringLiteralNode`, `BooleanLiteralNode`, `ComparisonNode`) |
| `expression_builder.go` | 291 | High-level `Build*` factory functions + `RenderCondition`, `RenderConditionAsIf` |
| `expression_optimizer.go` | 541 | Boolean-algebra optimizer (`OptimizeExpression`), called by `RenderCondition` |
| `expression_parser.go` | 510 | Recursive-descent parser (`ParseExpression`, `ExpressionParser`) |
| `expression_patterns.go` | 191 | **Centralized** public regex vars (`ExpressionPattern`, `NeedsStepsPattern`, `OrPattern`, etc.) |
| `expression_extraction.go` | 366 | `ExpressionExtractor`, env-var substitution, `SubstituteImportInputs` |
| `expression_safety_validation.go` | 302 | Security allowlist validation, **private** duplicate regexes |
| `expression_syntax_validation.go` | 236 | Structural validation (balanced braces, quotes, parentheses), **private** duplicate regexes |
| `known_needs_expressions.go` | 263 | `generateKnownNeedsExpressions`, `filterExpressionsForActivation`, `parseNeedsField` |
| **Total** | **2,896** | |

### Test Files

| File | Lines |
|------|------:|
| `expression_optimizer_test.go` | 1,210 |
| `expressions_test.go` | 1,144 |
| `expression_safety_test.go` | 849 |
| `expression_parser_comprehensive_test.go` | 621 |
| `expression_coverage_test.go` | 225 |
| `expressions_benchmark_test.go` | 183 |
| `expression_extraction_test.go` | ~200 |
| `expression_patterns_test.go` | ~100 |
| **Total** | **~4,532+** |

---

## Key Finding: Regex Duplication

`expression_patterns.go` was created to be the **single source of truth** for all expression regex
patterns. However, two files still carry **private duplicates** of patterns already defined there.

### Duplication Map — `expression_safety_validation.go`

| `expression_patterns.go` (public) | `expression_safety_validation.go` (private duplicate) |
|-----------------------------------|-------------------------------------------------------|
| `ExpressionPatternDotAll` | `expressionRegex` |
| `NeedsStepsPattern` | `needsStepsRegex` |
| `InputsPattern` | `inputsRegex` |
| `WorkflowCallInputsPattern` | `workflowCallInputsRegex` |
| `AWInputsPattern` | `awInputsRegex` |
| `AWImportInputsPattern` | `awImportInputsRegex` |
| `EnvPattern` | `envRegex` |
| `ComparisonExtractionPattern` | `comparisonExtractionRegex` |
| `OrPattern` | `orExpressionPattern` |

### Duplication Map — `expression_syntax_validation.go`

| `expression_patterns.go` (public) | `expression_syntax_validation.go` (private duplicate) |
|-----------------------------------|-------------------------------------------------------|
| `StringLiteralPattern` | `stringLiteralRegex` |
| `NumberLiteralPattern` | `numberLiteralRegex` |

### Unique Patterns in `expression_syntax_validation.go` (not yet centralized)

| Private name | Regex | Candidate public name |
|--------------|-------|-----------------------|
| `expressionBracesPattern` | `` `\$\{\{([^}]*)\}\}` `` | `ExpressionBracesPattern` |
| `exprPartSplitRe` | `` `[.\[\]]+` `` | `ExpressionPartSplitPattern` |
| `exprNumericPartRe` | `` `^\d+$` `` | `ExpressionNumericPartPattern` |

**Summary**: 9 regexes in `expression_safety_validation.go` and 2 regexes in
`expression_syntax_validation.go` duplicate patterns already in `expression_patterns.go`.
This means **double compilation overhead at startup** and **two sources of truth** that can drift.

---

## Regex Pattern Cross-Reference (Exact Strings)

### `ExpressionPatternDotAll` vs `expressionRegex`

```
expression_patterns.go:        `(?s)\$\{\{(.*?)\}\}`
expression_safety_validation:  `(?s)\$\{\{(.*?)\}\}`   ← identical
```

### `NeedsStepsPattern` vs `needsStepsRegex`

```
expression_patterns.go:        `^(needs|steps)\.[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)*$`
expression_safety_validation:  `^(needs|steps)\.[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)*$`  ← identical
```

### `ComparisonExtractionPattern` vs `comparisonExtractionRegex`

```
expression_patterns.go:        `([a-zA-Z_][a-zA-Z0-9_.]*)\s*(?:==|!=|<|>|<=|>=)\s*`
expression_safety_validation:  `([a-zA-Z_][a-zA-Z0-9_.]*)\s*(?:==|!=|<|>|<=|>=)\s*`  ← identical
```

### `StringLiteralPattern` vs `stringLiteralRegex`

```
expression_patterns.go:        `^'[^']*'$|^"[^"]*"$|^` + "`[^`]*`$"
expression_syntax_validation:  `^'[^']*'$|^"[^"]*"$|^` + "`[^`]*`$"  ← identical
```

### `NumberLiteralPattern` vs `numberLiteralRegex`

```
expression_patterns.go:        `^-?\d+(\.\d+)?$`
expression_syntax_validation:  `^-?\d+(\.\d+)?$`  ← identical
```

---

## Dependency Graph

```
expression_nodes.go          ← defines all AST types (no deps on other expression files)
       ↑
expression_builder.go        ← Build* constructors, RenderCondition → calls OptimizeExpression
       ↑
expression_optimizer.go      ← OptimizeExpression (pure tree transform, depends on nodes only)
       ↑
expression_parser.go         ← ParseExpression → produces ConditionNode trees

expression_patterns.go       ← standalone regex vars, no deps on other expression files

expression_extraction.go     ← uses ExpressionPatternDotAll, NeedsStepsPattern, etc.
                                ALSO defines ExpressionMapping, ExpressionExtractor

expression_safety_validation.go ← SHOULD use expression_patterns.go vars
                                   CURRENTLY has 9 private duplicate regexes

expression_syntax_validation.go ← SHOULD use expression_patterns.go vars
                                   CURRENTLY has 2 private duplicate regexes + 3 unique

known_needs_expressions.go   ← uses ExpressionMapping from expression_extraction.go
                                produces []ExpressionMapping for the compiler
```

---

## Type Inventory

### AST Node Types (`expression_nodes.go`)

| Type | Kind | Description |
|------|------|-------------|
| `ConditionNode` | interface | Single method: `Render() string` |
| `ExpressionNode` | leaf | Raw expression string + optional description |
| `AndNode` | binary | `&&` operator |
| `OrNode` | binary | `\|\|` operator |
| `NotNode` | unary | `!` operator |
| `DisjunctionNode` | n-ary OR | Avoids deep nesting; supports multiline render |
| `FunctionCallNode` | leaf | `name(args...)` |
| `PropertyAccessNode` | leaf | Dotted path like `github.event_name` |
| `StringLiteralNode` | leaf | `'value'` |
| `BooleanLiteralNode` | leaf | `true` / `false` |
| `ComparisonNode` | binary | `left op right` where op ∈ `==`, `!=`, `<`, `>`, `<=`, `>=` |

### Extractor Types (`expression_extraction.go`)

| Type | Description |
|------|-------------|
| `ExpressionMapping` | Maps a `${{ expr }}` to an env-var name + metadata |
| `ExpressionExtractor` | Stateful extractor with dedup map |

### Parser Types (`expression_parser.go`)

| Type | Description |
|------|-------------|
| `ExpressionParser` | Recursive-descent parser |
| `token` / `tokenKind` | Lexer tokens (private) |

### Validation Types (`expression_safety_validation.go`)

| Type | Description |
|------|-------------|
| `ExpressionValidationOptions` | Options struct for `validateSingleExpression` |

---

## Optimizer Rules (`expression_optimizer.go`, 541 lines)

| Rule | Node type | Before → After |
|------|-----------|----------------|
| Constant folding | `NotNode` | `!true → false`, `!false → true` |
| Double negation | `NotNode` | `!!A → A` |
| Boolean identity (AND) | `AndNode` | `A && true → A` |
| Boolean identity (OR) | `OrNode` | `A \|\| false → A` |
| Boolean annihilation (AND) | `AndNode` | `A && false → false` |
| Boolean annihilation (OR) | `OrNode` | `A \|\| true → true` |
| Idempotent (AND) | `AndNode` | `A && A → A` |
| Idempotent (OR) | `OrNode` | `A \|\| A → A` |
| Complement (AND) | `AndNode` | `A && !A → false` |
| Complement (OR) | `OrNode` | `A \|\| !A → true` |
| De Morgan (AND) | `NotNode` | `!(A && B) → !A \|\| !B` |
| De Morgan (OR) | `NotNode` | `!(A \|\| B) → !A && !B` |
| Absorption (AND) | `AndNode` | `A && (A \|\| B) → A` |
| Absorption (OR) | `OrNode` | `A \|\| (A && B) → A` |
| Subsumption | `DisjunctionNode` | `disj(A, A&&B, …) → disj(A, …)` |
| Deduplication | `AndNode` / `DisjunctionNode` | removes identical terms |
| False-filtering | `DisjunctionNode` | removes `false` terms |
| True short-circuit | `DisjunctionNode` | entire disjunction → `true` |

**Safety constraint**: No rule fires when a GitHub Actions status function
(`always()`, `success()`, `failure()`, `cancelled()`) appears in either operand.

**Termination**: Fixed-point iteration, bounded by `maxOptimizationPasses = 10`.

---

## Regex Compilation Count (startup cost)

| File | Compiled regexes | Notes |
|------|----------------:|-------|
| `expression_patterns.go` | 20 | Public, centralized |
| `expression_safety_validation.go` | 9 | All duplicates of patterns above |
| `expression_syntax_validation.go` | 5 | 2 duplicates + 3 unique |
| `expression_extraction.go` | 4 | Not duplicated |
| **Total compiled** | **~38** | |
| **Total unique patterns** | **~27** | |
| **Wasted compilations** | **11** | |

---

## Consolidation Plan

### Step 1 — `expression_safety_validation.go`: replace 9 private vars

| Remove | Replace with |
|--------|-------------|
| `expressionRegex` | `ExpressionPatternDotAll` |
| `needsStepsRegex` | `NeedsStepsPattern` |
| `inputsRegex` | `InputsPattern` |
| `workflowCallInputsRegex` | `WorkflowCallInputsPattern` |
| `awInputsRegex` | `AWInputsPattern` |
| `awImportInputsRegex` | `AWImportInputsPattern` |
| `envRegex` | `EnvPattern` |
| `comparisonExtractionRegex` | `ComparisonExtractionPattern` |
| `orExpressionPattern` | `OrPattern` |

### Step 2 — `expression_syntax_validation.go`: replace 2 private vars

| Remove | Replace with |
|--------|-------------|
| `stringLiteralRegex` | `StringLiteralPattern` |
| `numberLiteralRegex` | `NumberLiteralPattern` |

### Step 3 — `expression_patterns.go`: promote 3 unique patterns (optional)

| Add | Regex | Taken from |
|-----|-------|-----------|
| `ExpressionBracesPattern` | `` `\$\{\{([^}]*)\}\}` `` | `expression_syntax_validation.go` |
| `ExpressionPartSplitPattern` | `` `[.\[\]]+` `` | `expression_syntax_validation.go` |
| `ExpressionNumericPartPattern` | `` `^\d+$` `` | `expression_syntax_validation.go` |

### Files changed

| File | Action |
|------|--------|
| `expression_safety_validation.go` | Remove 9 private regex vars; update call sites |
| `expression_syntax_validation.go` | Remove 2 private regex vars; update call sites; optionally promote 3 unique patterns |
| `expression_patterns.go` | Optionally add 3 new public patterns |
| All `*_test.go` files | **No changes needed** |

---

## Risk Assessment

| Risk | Severity | Notes |
|------|----------|-------|
| Regex behavioral change | **None** | All duplicate patterns are byte-for-byte identical (verified above) |
| Startup perf change | **Positive** | From ~38 to ~27 compiled regexes |
| Test breakage | **None** | Internal implementation detail; tests use public API |
| Naming collision | **Low** | Names differ (`expressionRegex` vs `ExpressionPatternDotAll`) |
