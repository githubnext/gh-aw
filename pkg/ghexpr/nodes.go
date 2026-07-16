package ghexpr

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var nodesLog = logger.New("ghexpr:nodes")

// ConditionNode represents a node in a condition expression tree.
// All concrete types satisfy this interface; use type assertions to distinguish them.
type ConditionNode interface {
	Render() string
}

// Compile-time assertions: all ConditionNode implementation types must satisfy the interface.
var (
	_ ConditionNode = (*ExpressionNode)(nil)
	_ ConditionNode = (*AndNode)(nil)
	_ ConditionNode = (*OrNode)(nil)
	_ ConditionNode = (*NotNode)(nil)
	_ ConditionNode = (*DisjunctionNode)(nil)
	_ ConditionNode = (*FunctionCallNode)(nil)
	_ ConditionNode = (*PropertyAccessNode)(nil)
	_ ConditionNode = (*StringLiteralNode)(nil)
	_ ConditionNode = (*BooleanLiteralNode)(nil)
	_ ConditionNode = (*ComparisonNode)(nil)
)

// ExpressionNode represents a leaf expression — an opaque string that is valid
// GitHub Actions expression syntax (without the ${{ }} wrapper).
type ExpressionNode struct {
	Expression  string
	Description string // Optional comment/description for the expression
}

// Render implements ConditionNode.
func (e *ExpressionNode) Render() string {
	return e.Expression
}

// AndNode represents an AND operation between two conditions.
type AndNode struct {
	Left, Right ConditionNode
}

// needsParensAsAndOperand returns true when child must be wrapped in parentheses
// when it appears as an operand of an && expression.  Or-level nodes and opaque
// ExpressionNodes must be wrapped to preserve operator precedence.
// NotNode is also wrapped to prevent the leading ! from becoming a YAML type-tag
// indicator when the full expression is placed in an `if:` YAML value.
func needsParensAsAndOperand(child ConditionNode) bool {
	switch child.(type) {
	case *OrNode, *DisjunctionNode, *ExpressionNode, *NotNode:
		return true
	}
	return false
}

// Render implements ConditionNode.
func (a *AndNode) Render() string {
	leftStr := a.Left.Render()
	if needsParensAsAndOperand(a.Left) {
		leftStr = "(" + leftStr + ")"
	}
	rightStr := a.Right.Render()
	if needsParensAsAndOperand(a.Right) {
		rightStr = "(" + rightStr + ")"
	}
	return leftStr + " && " + rightStr
}

// OrNode represents an OR operation between two conditions.
//
// || has the lowest precedence of any boolean operator, so no child of an OR
// expression ever needs explicit parentheses to preserve evaluation order.
type OrNode struct {
	Left, Right ConditionNode
}

// Render implements ConditionNode.
func (o *OrNode) Render() string {
	return o.Left.Render() + " || " + o.Right.Render()
}

// NotNode represents a NOT operation on a condition.
type NotNode struct {
	Child ConditionNode
}

// Render implements ConditionNode.
func (n *NotNode) Render() string {
	// For simple function calls like cancelled(), render as !cancelled() instead of !(cancelled())
	// This prevents GitHub Actions from interpreting the extra parentheses as an object structure.
	if _, isFunctionCall := n.Child.(*FunctionCallNode); isFunctionCall {
		return "!" + n.Child.Render()
	}
	return "!(" + n.Child.Render() + ")"
}

// DisjunctionNode represents an OR operation with multiple terms to avoid deep nesting.
// When Multiline is true, each term is rendered on its own line.
type DisjunctionNode struct {
	Terms     []ConditionNode
	Multiline bool
}

// Render implements ConditionNode.
func (d *DisjunctionNode) Render() string {
	if len(d.Terms) == 0 {
		return ""
	}
	if len(d.Terms) == 1 {
		return d.Terms[0].Render()
	}

	if d.Multiline {
		return d.RenderMultiline()
	}

	nodesLog.Printf("Rendering inline disjunction with %d terms", len(d.Terms))
	var parts []string
	for _, term := range d.Terms {
		parts = append(parts, term.Render())
	}
	return strings.Join(parts, " || ")
}

// RenderMultiline renders the disjunction with each term on a separate line,
// including comments for ExpressionNode values that carry a Description.
func (d *DisjunctionNode) RenderMultiline() string {
	if len(d.Terms) == 0 {
		return ""
	}
	if len(d.Terms) == 1 {
		return d.Terms[0].Render()
	}

	nodesLog.Printf("Rendering multiline disjunction with %d terms", len(d.Terms))

	var lines []string
	for i, term := range d.Terms {
		var line string

		if expr, ok := term.(*ExpressionNode); ok && expr.Description != "" {
			line = "# " + expr.Description + "\n"
		}

		if i < len(d.Terms)-1 {
			line += term.Render() + " ||"
		} else {
			line += term.Render()
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// FunctionCallNode represents a function call such as contains(array, value).
type FunctionCallNode struct {
	FunctionName string
	Arguments    []ConditionNode
}

// Render implements ConditionNode.
func (f *FunctionCallNode) Render() string {
	var args []string
	for _, arg := range f.Arguments {
		args = append(args, arg.Render())
	}
	return f.FunctionName + "(" + strings.Join(args, ", ") + ")"
}

// PropertyAccessNode represents a property-access path such as github.event.action.
type PropertyAccessNode struct {
	PropertyPath string
}

// Render implements ConditionNode.
func (p *PropertyAccessNode) Render() string {
	return p.PropertyPath
}

// StringLiteralNode represents a single-quoted string literal.
// Embedded single quotes are escaped by doubling them per the GitHub Actions spec.
type StringLiteralNode struct {
	Value string
}

// Render implements ConditionNode.
func (s *StringLiteralNode) Render() string {
	escaped := strings.ReplaceAll(s.Value, "'", "''")
	return "'" + escaped + "'"
}

// BooleanLiteralNode represents the literals true or false.
type BooleanLiteralNode struct {
	Value bool
}

// Render implements ConditionNode.
func (b *BooleanLiteralNode) Render() string {
	if b.Value {
		return "true"
	}
	return "false"
}

// ComparisonNode represents a binary comparison: ==, !=, <, >, <=, or >=.
type ComparisonNode struct {
	Left     ConditionNode
	Operator string
	Right    ConditionNode
}

// Render implements ConditionNode.
func (c *ComparisonNode) Render() string {
	return c.Left.Render() + " " + c.Operator + " " + c.Right.Render()
}
