// Package ghexpr implements a parser, optimizer, and builder for GitHub Actions
// expression syntax (${{ ... }}).
//
// # Grammar
//
// The following grammar describes the subset of GitHub Actions expression
// syntax handled by this package (adapted from the official spec at
// https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/evaluate-expressions-in-workflows-and-actions):
//
//	expression     = or_expr
//	or_expr        = and_expr ( "||" and_expr )*
//	and_expr       = unary_expr ( "&&" unary_expr )*
//	unary_expr     = "!" unary_expr | primary_expr
//	primary_expr   = "(" expression ")"
//	               | function_call
//	               | literal
//	               | property_access
//	function_call  = IDENT "(" [ expression ( "," expression )* ] ")"
//	property_access= IDENT ( "." IDENT )*
//	               | IDENT "[" expression "]"
//	literal        = STRING | NUMBER | "true" | "false" | "null"
//	STRING         = "'" [^']* "'" | '"' [^"]* '"'
//	NUMBER         = "-"? DIGIT+ ( "." DIGIT+ )?
//	IDENT          = [a-zA-Z_] [a-zA-Z0-9_-]*
//
// Note: GitHub Actions expressions are always wrapped in ${{ and }} markers
// when embedded in YAML values.  This package operates on the inner content
// (without the ${{ }} wrappers) unless otherwise documented.
//
// # AST
//
// Parsed expressions are represented as a tree of [ConditionNode] values:
//
//   - [ExpressionNode]    – an opaque leaf expression
//   - [AndNode]           – left && right
//   - [OrNode]            – left || right
//   - [NotNode]           – !child
//   - [ComparisonNode]    – left op right  (==, !=, <, >, <=, >=)
//   - [FunctionCallNode]  – func(arg, …)
//   - [PropertyAccessNode]– a.b.c property path
//   - [StringLiteralNode] – 'value'
//   - [BooleanLiteralNode]– true / false
//   - [DisjunctionNode]   – multi-term OR, for rendering as GitHub's multi-line
//
// # Optimizer
//
// [OptimizeExpression] applies boolean-algebra simplifications (idempotent law,
// absorption, De Morgan, etc.) to produce a shorter equivalent expression.
// Status-function calls (always, success, failure, cancelled) are treated as
// opaque so the optimizer never removes them.
//
// # Builders
//
// Composable [Build…] functions construct condition trees without touching raw
// strings.  [RenderCondition] renders a tree to its GitHub Actions syntax.
//
// # Patterns
//
// Pre-compiled [regexp.Regexp] values are exposed for common matching tasks:
//
//   - [ExpressionPattern]         – matches ${{ … }}
//   - [ExpressionPatternDotAll]   – same with dot-all flag
//   - [StringLiteralPattern]      – matches 'x', "x", or `x`
//   - [NumberLiteralPattern]      – matches numeric literals
//   - [OrPattern]                 – splits left || right
//   - [ComparisonExtractionPattern] – extracts the left side of comparisons
//   - [InlineExpressionPattern]   – matches inline ${{ … }}
//   - [TemplateIfPattern]         – matches {{#if … }}
//   - [TemplateElseIfPattern]     – matches {{#elseif … }}
//   - [RangePattern]              – matches numeric ranges (e.g. "1-10")
//
// Predicate helpers: [HasExpressionMarker], [ContainsExpression], [IsExpression].
package ghexpr
