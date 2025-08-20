package workflow

import (
	"fmt"
	"strings"
)

// ConditionNode represents a node in a condition expression tree
type ConditionNode interface {
	Render() string
}

// ExpressionNode represents a leaf expression
type ExpressionNode struct {
	Expression  string
	Description string // Optional comment/description for the expression
}

func (e *ExpressionNode) Render() string {
	return e.Expression
}

// AndNode represents an AND operation between two conditions
type AndNode struct {
	Left, Right ConditionNode
}

func (a *AndNode) Render() string {
	return fmt.Sprintf("(%s) && (%s)", a.Left.Render(), a.Right.Render())
}

// OrNode represents an OR operation between two conditions
type OrNode struct {
	Left, Right ConditionNode
}

func (o *OrNode) Render() string {
	return fmt.Sprintf("(%s) || (%s)", o.Left.Render(), o.Right.Render())
}

// NotNode represents a NOT operation on a condition
type NotNode struct {
	Child ConditionNode
}

func (n *NotNode) Render() string {
	return fmt.Sprintf("!(%s)", n.Child.Render())
}

// DisjunctionNode represents an OR operation with multiple terms to avoid deep nesting
type DisjunctionNode struct {
	Terms     []ConditionNode
	Multiline bool // If true, render each term on separate line with comments
}

func (d *DisjunctionNode) Render() string {
	if len(d.Terms) == 0 {
		return ""
	}
	if len(d.Terms) == 1 {
		return d.Terms[0].Render()
	}

	// Use multiline rendering if enabled
	if d.Multiline {
		return d.RenderMultiline()
	}

	var parts []string
	for _, term := range d.Terms {
		parts = append(parts, term.Render())
	}
	return strings.Join(parts, " || ")
}

// RenderMultiline renders the disjunction with each term on a separate line,
// including comments for expressions that have descriptions
func (d *DisjunctionNode) RenderMultiline() string {
	if len(d.Terms) == 0 {
		return ""
	}
	if len(d.Terms) == 1 {
		return d.Terms[0].Render()
	}

	var lines []string
	for i, term := range d.Terms {
		var line string

		// Add comment if this is an ExpressionNode with a description
		if expr, ok := term.(*ExpressionNode); ok && expr.Description != "" {
			line = "# " + expr.Description + "\n"
		}

		// Add the expression with OR operator (except for the last term)
		if i < len(d.Terms)-1 {
			line += term.Render() + " ||"
		} else {
			line += term.Render()
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// FunctionCallNode represents a function call expression like contains(array, value)
type FunctionCallNode struct {
	FunctionName string
	Arguments    []ConditionNode
}

func (f *FunctionCallNode) Render() string {
	var args []string
	for _, arg := range f.Arguments {
		args = append(args, arg.Render())
	}
	return fmt.Sprintf("%s(%s)", f.FunctionName, strings.Join(args, ", "))
}

// PropertyAccessNode represents property access like github.event.action
type PropertyAccessNode struct {
	PropertyPath string
}

func (p *PropertyAccessNode) Render() string {
	return p.PropertyPath
}

// StringLiteralNode represents a string literal value
type StringLiteralNode struct {
	Value string
}

func (s *StringLiteralNode) Render() string {
	return fmt.Sprintf("'%s'", s.Value)
}

// BooleanLiteralNode represents a boolean literal value
type BooleanLiteralNode struct {
	Value bool
}

func (b *BooleanLiteralNode) Render() string {
	if b.Value {
		return "true"
	}
	return "false"
}

// NumberLiteralNode represents a numeric literal value
type NumberLiteralNode struct {
	Value string
}

func (n *NumberLiteralNode) Render() string {
	return n.Value
}

// ComparisonNode represents comparison operations like ==, !=, <, >, <=, >=
type ComparisonNode struct {
	Left     ConditionNode
	Operator string
	Right    ConditionNode
}

func (c *ComparisonNode) Render() string {
	return fmt.Sprintf("%s %s %s", c.Left.Render(), c.Operator, c.Right.Render())
}

// TernaryNode represents ternary conditional expressions like condition ? true_value : false_value
type TernaryNode struct {
	Condition  ConditionNode
	TrueValue  ConditionNode
	FalseValue ConditionNode
}

func (t *TernaryNode) Render() string {
	return fmt.Sprintf("%s ? %s : %s", t.Condition.Render(), t.TrueValue.Render(), t.FalseValue.Render())
}

// ContainsNode represents array membership checks using contains() function
type ContainsNode struct {
	Array ConditionNode
	Value ConditionNode
}

func (c *ContainsNode) Render() string {
	return fmt.Sprintf("contains(%s, %s)", c.Array.Render(), c.Value.Render())
}

// buildConditionTree creates a condition tree from existing if condition and new draft condition
func buildConditionTree(existingCondition string, draftCondition string) ConditionNode {
	draftNode := &ExpressionNode{Expression: draftCondition}

	if existingCondition == "" {
		return draftNode
	}

	existingNode := &ExpressionNode{Expression: existingCondition}
	return &AndNode{Left: existingNode, Right: draftNode}
}

// buildReactionCondition creates a condition tree for the add_reaction job
func buildReactionCondition() ConditionNode {
	// Build a list of event types that should trigger reactions using the new expression nodes
	var terms []ConditionNode

	terms = append(terms, BuildEventTypeEquals("issues"))
	terms = append(terms, BuildEventTypeEquals("pull_request"))
	terms = append(terms, BuildEventTypeEquals("issue_comment"))
	terms = append(terms, BuildEventTypeEquals("pull_request_comment"))
	terms = append(terms, BuildEventTypeEquals("pull_request_review_comment"))

	// Use DisjunctionNode to avoid deep nesting
	return &DisjunctionNode{Terms: terms}
}

// Helper functions for building common GitHub Actions expression patterns

// BuildPropertyAccess creates a property access node for GitHub context properties
func BuildPropertyAccess(path string) *PropertyAccessNode {
	return &PropertyAccessNode{PropertyPath: path}
}

// BuildStringLiteral creates a string literal node
func BuildStringLiteral(value string) *StringLiteralNode {
	return &StringLiteralNode{Value: value}
}

// BuildBooleanLiteral creates a boolean literal node
func BuildBooleanLiteral(value bool) *BooleanLiteralNode {
	return &BooleanLiteralNode{Value: value}
}

// BuildNumberLiteral creates a number literal node
func BuildNumberLiteral(value string) *NumberLiteralNode {
	return &NumberLiteralNode{Value: value}
}

// BuildComparison creates a comparison node with the specified operator
func BuildComparison(left ConditionNode, operator string, right ConditionNode) *ComparisonNode {
	return &ComparisonNode{Left: left, Operator: operator, Right: right}
}

// BuildEquals creates an equality comparison
func BuildEquals(left ConditionNode, right ConditionNode) *ComparisonNode {
	return BuildComparison(left, "==", right)
}

// BuildNotEquals creates an inequality comparison
func BuildNotEquals(left ConditionNode, right ConditionNode) *ComparisonNode {
	return BuildComparison(left, "!=", right)
}

// BuildContains creates a contains() function call node
func BuildContains(array ConditionNode, value ConditionNode) *ContainsNode {
	return &ContainsNode{Array: array, Value: value}
}

// BuildFunctionCall creates a function call node
func BuildFunctionCall(functionName string, args ...ConditionNode) *FunctionCallNode {
	return &FunctionCallNode{FunctionName: functionName, Arguments: args}
}

// BuildTernary creates a ternary conditional expression
func BuildTernary(condition ConditionNode, trueValue ConditionNode, falseValue ConditionNode) *TernaryNode {
	return &TernaryNode{Condition: condition, TrueValue: trueValue, FalseValue: falseValue}
}

// BuildLabelContains creates a condition to check if an issue/PR contains a specific label
func BuildLabelContains(labelName string) *ContainsNode {
	return BuildContains(
		BuildPropertyAccess("github.event.issue.labels.*.name"),
		BuildStringLiteral(labelName),
	)
}

// BuildActionEquals creates a condition to check if the event action equals a specific value
func BuildActionEquals(action string) *ComparisonNode {
	return BuildEquals(
		BuildPropertyAccess("github.event.action"),
		BuildStringLiteral(action),
	)
}

// BuildEventTypeEquals creates a condition to check if the event type equals a specific value
func BuildEventTypeEquals(eventType string) *ComparisonNode {
	return BuildEquals(
		BuildPropertyAccess("github.event_name"),
		BuildStringLiteral(eventType),
	)
}

// BuildRefStartsWith creates a condition to check if github.ref starts with a prefix
func BuildRefStartsWith(prefix string) *FunctionCallNode {
	return BuildFunctionCall("startsWith",
		BuildPropertyAccess("github.ref"),
		BuildStringLiteral(prefix),
	)
}

// BuildExpressionWithDescription creates an expression node with an optional description
func BuildExpressionWithDescription(expression, description string) *ExpressionNode {
	return &ExpressionNode{
		Expression:  expression,
		Description: description,
	}
}

// BuildMultilineDisjunction creates a disjunction node with multiline rendering enabled
func BuildMultilineDisjunction(terms ...ConditionNode) *DisjunctionNode {
	return &DisjunctionNode{
		Terms:     terms,
		Multiline: true,
	}
}
