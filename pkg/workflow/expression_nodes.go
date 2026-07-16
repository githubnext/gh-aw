// This file re-exports the GitHub Actions expression AST types from pkg/ghexpr
// as type aliases so that all existing code in pkg/workflow continues to compile
// without modification.  New code should import pkg/ghexpr directly.
package workflow

import "github.com/github/gh-aw/pkg/ghexpr"

// ConditionNode re-exports [ghexpr.ConditionNode].
type ConditionNode = ghexpr.ConditionNode

// ExpressionNode re-exports [ghexpr.ExpressionNode].
type ExpressionNode = ghexpr.ExpressionNode

// AndNode re-exports [ghexpr.AndNode].
type AndNode = ghexpr.AndNode

// OrNode re-exports [ghexpr.OrNode].
type OrNode = ghexpr.OrNode

// NotNode re-exports [ghexpr.NotNode].
type NotNode = ghexpr.NotNode

// DisjunctionNode re-exports [ghexpr.DisjunctionNode].
type DisjunctionNode = ghexpr.DisjunctionNode

// FunctionCallNode re-exports [ghexpr.FunctionCallNode].
type FunctionCallNode = ghexpr.FunctionCallNode

// PropertyAccessNode re-exports [ghexpr.PropertyAccessNode].
type PropertyAccessNode = ghexpr.PropertyAccessNode

// StringLiteralNode re-exports [ghexpr.StringLiteralNode].
type StringLiteralNode = ghexpr.StringLiteralNode

// BooleanLiteralNode re-exports [ghexpr.BooleanLiteralNode].
type BooleanLiteralNode = ghexpr.BooleanLiteralNode

// ComparisonNode re-exports [ghexpr.ComparisonNode].
type ComparisonNode = ghexpr.ComparisonNode
