package ghexpr

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var builderLog = logger.New("ghexpr:builder")

// BuildConditionTree creates a condition tree from an existing condition string and a
// new draft condition string.  When existingCondition is empty, the draft is returned
// as a standalone [ExpressionNode]; otherwise the two are combined with AND.
func BuildConditionTree(existingCondition string, draftCondition string) ConditionNode {
	builderLog.Printf("Building condition tree: existing=%q, draft=%q", existingCondition, draftCondition)
	draftNode := &ExpressionNode{Expression: draftCondition}
	if existingCondition == "" {
		return draftNode
	}
	existingNode := &ExpressionNode{Expression: existingCondition}
	return &AndNode{Left: existingNode, Right: draftNode}
}

// BuildOr creates an OR node combining two conditions.
func BuildOr(left ConditionNode, right ConditionNode) ConditionNode {
	return &OrNode{Left: left, Right: right}
}

// BuildAnd creates an AND node combining two conditions.
func BuildAnd(left ConditionNode, right ConditionNode) ConditionNode {
	builderLog.Print("Building AND condition node")
	return &AndNode{Left: left, Right: right}
}

// BuildPropertyAccess creates a property-access node for a dotted path.
// Example: BuildPropertyAccess("github.event.action")
func BuildPropertyAccess(path string) *PropertyAccessNode {
	return &PropertyAccessNode{PropertyPath: path}
}

// BuildStringLiteral creates a single-quoted string literal node.
func BuildStringLiteral(value string) *StringLiteralNode {
	return &StringLiteralNode{Value: value}
}

// BuildBooleanLiteral creates a boolean literal node (true or false).
func BuildBooleanLiteral(value bool) *BooleanLiteralNode {
	return &BooleanLiteralNode{Value: value}
}

// BuildNullLiteral creates a null literal node.
func BuildNullLiteral() *ExpressionNode {
	return &ExpressionNode{Expression: "null"}
}

// BuildComparison creates a comparison node with an explicit operator string
// such as "==", "!=", "<", ">", "<=", or ">=".
func BuildComparison(left ConditionNode, operator string, right ConditionNode) *ComparisonNode {
	return &ComparisonNode{Left: left, Operator: operator, Right: right}
}

// BuildEquals creates an equality comparison node (==).
func BuildEquals(left ConditionNode, right ConditionNode) *ComparisonNode {
	return BuildComparison(left, "==", right)
}

// BuildNotEquals creates an inequality comparison node (!=).
func BuildNotEquals(left ConditionNode, right ConditionNode) *ComparisonNode {
	return BuildComparison(left, "!=", right)
}

// BuildFunctionCall creates a function-call node.
// Example: BuildFunctionCall("contains", BuildPropertyAccess("github.event_name"), BuildStringLiteral("push"))
func BuildFunctionCall(functionName string, args ...ConditionNode) *FunctionCallNode {
	return &FunctionCallNode{FunctionName: functionName, Arguments: args}
}

// BuildDisjunction creates a multi-term OR node.
// When multiline is true, [DisjunctionNode.RenderMultiline] is used.
func BuildDisjunction(multiline bool, terms ...ConditionNode) *DisjunctionNode {
	return &DisjunctionNode{Terms: terms, Multiline: multiline}
}

// RenderCondition runs [OptimizeExpression] on node and returns the rendered string.
func RenderCondition(node ConditionNode) string {
	return OptimizeExpression(node).Render()
}

// RenderConditionAsIf writes a YAML `if:` block scalar entry to yaml using the given
// indentation prefix for each line of the rendered condition.  The condition is first
// optimized with [OptimizeExpression].
func RenderConditionAsIf(yaml *strings.Builder, condition ConditionNode, indent string) {
	yaml.WriteString("        if: |\n")
	conditionStr := RenderCondition(condition)
	for line := range strings.SplitSeq(conditionStr, "\n") {
		yaml.WriteString(indent + line + "\n")
	}
}
