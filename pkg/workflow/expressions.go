package workflow

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var expressionsLog = logger.New("workflow:expressions")

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
	// For simple function calls like cancelled(), render as !cancelled() instead of !(cancelled())
	// This prevents GitHub Actions from interpreting the extra parentheses as an object structure
	if _, isFunctionCall := n.Child.(*FunctionCallNode); isFunctionCall {
		return fmt.Sprintf("!%s", n.Child.Render())
	}
	return fmt.Sprintf("!(%s)", n.Child.Render())
}

// ParenthesesNode wraps a condition in parentheses for proper YAML interpretation
type ParenthesesNode struct {
	Child ConditionNode
}

func (p *ParenthesesNode) Render() string {
	return fmt.Sprintf("(%s)", p.Child.Render())
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

func buildOr(left ConditionNode, right ConditionNode) ConditionNode {
	return &OrNode{Left: left, Right: right}
}

func buildAnd(left ConditionNode, right ConditionNode) ConditionNode {
	return &AndNode{Left: left, Right: right}
}

// buildReactionCondition creates a condition tree for the add_reaction job
func buildReactionCondition() ConditionNode {
	// Build a list of event types that should trigger reactions using the new expression nodes
	var terms []ConditionNode

	terms = append(terms, BuildEventTypeEquals("issues"))
	terms = append(terms, BuildEventTypeEquals("issue_comment"))
	terms = append(terms, BuildEventTypeEquals("pull_request_review_comment"))
	terms = append(terms, BuildEventTypeEquals("discussion"))
	terms = append(terms, BuildEventTypeEquals("discussion_comment"))

	// For pull_request events, we need to ensure it's not from a forked repository
	// since forked repositories have read-only permissions and cannot add reactions
	pullRequestCondition := &AndNode{
		Left:  BuildEventTypeEquals("pull_request"),
		Right: BuildNotFromFork(),
	}
	terms = append(terms, pullRequestCondition)

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

// BuildNullLiteral creates a null literal node
func BuildNullLiteral() *ExpressionNode {
	return &ExpressionNode{Expression: "null"}
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

// BuildNotFromFork creates a condition to check that a pull request is not from a forked repository
// This prevents the job from running on forked PRs where write permissions are not available
func BuildNotFromFork() *ComparisonNode {
	return BuildEquals(
		BuildPropertyAccess("github.event.pull_request.head.repo.full_name"),
		BuildPropertyAccess("github.repository"),
	)
}

func BuildSafeOutputType(outputType string, min int) ConditionNode {
	// Use !cancelled() && needs.agent.result != 'skipped' to properly handle workflow cancellation
	// !cancelled() allows jobs to run when dependencies fail (for error reporting)
	// needs.agent.result != 'skipped' prevents running when workflow is cancelled (dependencies get skipped)
	notCancelledFunc := &NotNode{
		Child: BuildFunctionCall("cancelled"),
	}

	// Check that agent job was not skipped (happens when workflow is cancelled)
	agentNotSkipped := &ComparisonNode{
		Left:     BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		Operator: "!=",
		Right:    BuildStringLiteral("skipped"),
	}

	// Combine !cancelled() with agent not skipped check
	baseCondition := &AndNode{
		Left:  notCancelledFunc,
		Right: agentNotSkipped,
	}

	// If min > 0, only return base condition without the contains check
	// This is needed to ensure the job runs even with 0 outputs to enforce the minimum constraint
	// Wrap in parentheses to ensure proper YAML interpretation
	if min > 0 {
		return &ParenthesesNode{Child: baseCondition}
	}

	containsFunc := BuildFunctionCall("contains",
		BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.output_types", constants.AgentJobName)),
		BuildStringLiteral(outputType),
	)
	return &AndNode{
		Left:  baseCondition,
		Right: containsFunc,
	}
}

// BuildFromAllowedForks creates a condition to check if a pull request is from an allowed fork
// Supports glob patterns like "org/*" and exact matches like "org/repo"
func BuildFromAllowedForks(allowedForks []string) ConditionNode {
	if len(allowedForks) == 0 {
		return BuildNotFromFork()
	}

	var conditions []ConditionNode

	// Always allow PRs from the same repository
	conditions = append(conditions, BuildNotFromFork())

	for _, pattern := range allowedForks {
		if strings.HasSuffix(pattern, "/*") {
			// Glob pattern: org/* matches org/anything
			prefix := strings.TrimSuffix(pattern, "*")
			condition := &FunctionCallNode{
				FunctionName: "startsWith",
				Arguments: []ConditionNode{
					BuildPropertyAccess("github.event.pull_request.head.repo.full_name"),
					BuildStringLiteral(prefix),
				},
			}
			conditions = append(conditions, condition)
		} else {
			// Exact match: org/repo
			condition := BuildEquals(
				BuildPropertyAccess("github.event.pull_request.head.repo.full_name"),
				BuildStringLiteral(pattern),
			)
			conditions = append(conditions, condition)
		}
	}

	if len(conditions) == 1 {
		return conditions[0]
	}

	// Use DisjunctionNode to combine all conditions with OR
	return &DisjunctionNode{Terms: conditions}
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

// stripExpressionWrapper removes the ${{ }} wrapper from an expression if present
func stripExpressionWrapper(expression string) string {
	// Trim whitespace
	expr := strings.TrimSpace(expression)
	// Check if it starts with ${{ and ends with }}
	if strings.HasPrefix(expr, "${{") && strings.HasSuffix(expr, "}}") {
		// Remove the wrapper and trim inner whitespace
		return strings.TrimSpace(expr[3 : len(expr)-2])
	}
	return expr
}

// BuildDisjunction creates a disjunction node (OR operation) from the given terms
// Handles arrays of size 0, 1, or more correctly
// The multiline parameter controls whether to render each term on a separate line
func BuildDisjunction(multiline bool, terms ...ConditionNode) *DisjunctionNode {
	return &DisjunctionNode{
		Terms:     terms,
		Multiline: multiline,
	}
}

// BuildPRCommentCondition creates a condition to check if the event is a comment on a pull request
// This checks for:
// - issue_comment on a PR (github.event.issue.pull_request != null)
// - pull_request_review_comment
// - pull_request_review
func BuildPRCommentCondition() ConditionNode {
	// issue_comment event on a PR
	issueCommentOnPR := buildAnd(
		BuildEventTypeEquals("issue_comment"),
		BuildComparison(
			BuildPropertyAccess("github.event.issue.pull_request"),
			"!=",
			&ExpressionNode{Expression: "null"},
		),
	)

	// pull_request_review_comment event
	prReviewComment := BuildEventTypeEquals("pull_request_review_comment")

	// pull_request_review event
	prReview := BuildEventTypeEquals("pull_request_review")

	// Combine all conditions with OR
	return &DisjunctionNode{
		Terms: []ConditionNode{
			issueCommentOnPR,
			prReviewComment,
			prReview,
		},
	}
}

// RenderConditionAsIf renders a ConditionNode as an 'if' condition with proper YAML indentation
func RenderConditionAsIf(yaml *strings.Builder, condition ConditionNode, indent string) {
	yaml.WriteString("        if: |\n")
	conditionStr := condition.Render()

	// Format the condition with proper indentation
	lines := strings.Split(conditionStr, "\n")
	for _, line := range lines {
		yaml.WriteString(indent + line + "\n")
	}
}

// ExpressionParser handles parsing of expression strings into ConditionNode trees
type ExpressionParser struct {
	tokens []token
	pos    int
}

type token struct {
	kind  tokenKind
	value string
	pos   int
}

type tokenKind int

const (
	tokenLiteral tokenKind = iota
	tokenAnd
	tokenOr
	tokenNot
	tokenLeftParen
	tokenRightParen
	tokenEOF
)

// ParseExpression parses a string expression into a ConditionNode tree
// Supports && (AND), || (OR), ! (NOT), and parentheses for grouping
// Example: "condition1 && (condition2 || !condition3)"
func ParseExpression(expression string) (ConditionNode, error) {
	expressionsLog.Printf("Parsing expression: %s", expression)

	if strings.TrimSpace(expression) == "" {
		return nil, fmt.Errorf("empty expression")
	}

	parser := &ExpressionParser{}
	tokens, err := parser.tokenize(expression)
	if err != nil {
		expressionsLog.Printf("Failed to tokenize expression: %v", err)
		return nil, err
	}
	parser.tokens = tokens
	parser.pos = 0

	result, err := parser.parseOrExpression()
	if err != nil {
		expressionsLog.Printf("Failed to parse expression: %v", err)
		return nil, err
	}

	// Check that all tokens were consumed
	if parser.current().kind != tokenEOF {
		return nil, fmt.Errorf("unexpected token '%s' at position %d", parser.current().value, parser.current().pos)
	}

	expressionsLog.Printf("Successfully parsed expression with %d tokens", len(tokens))
	return result, nil
}

// tokenize breaks the expression string into tokens
func (p *ExpressionParser) tokenize(expression string) ([]token, error) {
	var tokens []token
	i := 0

	for i < len(expression) {
		// Skip whitespace
		if unicode.IsSpace(rune(expression[i])) {
			i++
			continue
		}

		switch {
		case i+1 < len(expression) && expression[i:i+2] == "&&":
			tokens = append(tokens, token{tokenAnd, "&&", i})
			i += 2
		case i+1 < len(expression) && expression[i:i+2] == "||":
			tokens = append(tokens, token{tokenOr, "||", i})
			i += 2
		case expression[i] == '!' && (i+1 >= len(expression) || expression[i+1] != '='):
			// Only treat ! as NOT if not followed by = (to avoid conflicting with !=)
			tokens = append(tokens, token{tokenNot, "!", i})
			i++
		case expression[i] == '(':
			tokens = append(tokens, token{tokenLeftParen, "(", i})
			i++
		case expression[i] == ')':
			tokens = append(tokens, token{tokenRightParen, ")", i})
			i++
		default:
			// Parse literal expression - everything until we hit a logical operator or paren
			start := i
			parenCount := 0

			for i < len(expression) {
				ch := expression[i]

				// Handle quoted strings - skip everything inside quotes
				if ch == '\'' || ch == '"' {
					quote := ch
					i++ // skip opening quote
					for i < len(expression) {
						if expression[i] == quote {
							i++ // skip closing quote
							break
						}
						if expression[i] == '\\' && i+1 < len(expression) {
							i += 2 // skip escaped character
						} else {
							i++
						}
					}
					continue
				}

				// Track parentheses that are part of the expression (e.g., function calls)
				if ch == '(' {
					parenCount++
					i++
					continue
				} else if ch == ')' {
					if parenCount > 0 {
						parenCount--
						i++
						continue
					} else {
						// This closes our group expression, stop here
						break
					}
				}

				// Check for logical operators when not inside parentheses
				if parenCount == 0 {
					// Check for && or ||
					if i+1 < len(expression) {
						next := expression[i : i+2]
						if next == "&&" || next == "||" {
							break
						}
					}

					// Check for logical NOT that's not part of !=
					if ch == '!' && (i+1 >= len(expression) || expression[i+1] != '=') {
						break
					}
				}

				i++
			}

			literal := strings.TrimSpace(expression[start:i])
			if literal == "" {
				return nil, fmt.Errorf("unexpected empty literal at position %d", start)
			}
			tokens = append(tokens, token{tokenLiteral, literal, start})
		}
	}

	tokens = append(tokens, token{tokenEOF, "", i})
	return tokens, nil
}

// parseOrExpression parses OR expressions (lowest precedence)
func (p *ExpressionParser) parseOrExpression() (ConditionNode, error) {
	left, err := p.parseAndExpression()
	if err != nil {
		return nil, err
	}

	for p.current().kind == tokenOr {
		p.advance() // consume ||
		right, err := p.parseAndExpression()
		if err != nil {
			return nil, err
		}
		left = &OrNode{Left: left, Right: right}
	}

	return left, nil
}

// parseAndExpression parses AND expressions (higher precedence than OR)
func (p *ExpressionParser) parseAndExpression() (ConditionNode, error) {
	left, err := p.parseUnaryExpression()
	if err != nil {
		return nil, err
	}

	for p.current().kind == tokenAnd {
		p.advance() // consume &&
		right, err := p.parseUnaryExpression()
		if err != nil {
			return nil, err
		}
		left = &AndNode{Left: left, Right: right}
	}

	return left, nil
}

// parseUnaryExpression parses NOT expressions and primary expressions
func (p *ExpressionParser) parseUnaryExpression() (ConditionNode, error) {
	if p.current().kind == tokenNot {
		p.advance() // consume !
		operand, err := p.parseUnaryExpression()
		if err != nil {
			return nil, err
		}
		return &NotNode{Child: operand}, nil
	}

	return p.parsePrimaryExpression()
}

// parsePrimaryExpression parses literals and parenthesized expressions
func (p *ExpressionParser) parsePrimaryExpression() (ConditionNode, error) {
	switch p.current().kind {
	case tokenLeftParen:
		p.advance() // consume (
		expr, err := p.parseOrExpression()
		if err != nil {
			return nil, err
		}
		if p.current().kind != tokenRightParen {
			return nil, fmt.Errorf("expected ')' at position %d", p.current().pos)
		}
		p.advance() // consume )
		return expr, nil

	case tokenLiteral:
		literal := p.current().value
		p.advance()
		return &ExpressionNode{Expression: literal}, nil

	default:
		return nil, fmt.Errorf("unexpected token '%s' at position %d", p.current().value, p.current().pos)
	}
}

// current returns the current token
func (p *ExpressionParser) current() token {
	if p.pos >= len(p.tokens) {
		return token{tokenEOF, "", -1}
	}
	return p.tokens[p.pos]
}

// advance moves to the next token
func (p *ExpressionParser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// VisitExpressionTree walks through an expression tree and calls the visitor function
// for each ExpressionNode (literal expression) found in the tree
func VisitExpressionTree(node ConditionNode, visitor func(expr *ExpressionNode) error) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ExpressionNode:
		return visitor(n)
	case *AndNode:
		if err := VisitExpressionTree(n.Left, visitor); err != nil {
			return err
		}
		return VisitExpressionTree(n.Right, visitor)
	case *OrNode:
		if err := VisitExpressionTree(n.Left, visitor); err != nil {
			return err
		}
		return VisitExpressionTree(n.Right, visitor)
	case *NotNode:
		return VisitExpressionTree(n.Child, visitor)
	case *DisjunctionNode:
		for _, term := range n.Terms {
			if err := VisitExpressionTree(term, visitor); err != nil {
				return err
			}
		}
	default:
		// For other node types (ComparisonNode, PropertyAccessNode, etc.)
		// we don't recurse since they represent complete literal expressions
		return nil
	}

	return nil
}

// BreakLongExpression breaks a long expression into multiple lines at logical points
// such as after || and && operators for better readability
func BreakLongExpression(expression string) []string {
	// If the expression is not too long, return as-is
	if len(expression) <= constants.MaxExpressionLineLength {
		return []string{expression}
	}

	expressionsLog.Printf("Breaking long expression: length=%d", len(expression))

	var lines []string
	current := ""
	i := 0

	for i < len(expression) {
		char := expression[i]

		// Handle quoted strings - don't break inside quotes
		if char == '\'' || char == '"' {
			quote := char
			current += string(char)
			i++

			// Continue until closing quote
			for i < len(expression) {
				current += string(expression[i])
				if expression[i] == quote {
					i++
					break
				}
				if expression[i] == '\\' && i+1 < len(expression) {
					i++ // Skip escaped character
					if i < len(expression) {
						current += string(expression[i])
					}
				}
				i++
			}
			continue
		}

		// Look for logical operators as break points
		if i+2 <= len(expression) {
			next2 := expression[i : i+2]
			if next2 == "||" || next2 == "&&" {
				current += next2
				i += 2

				// If the current line is getting long (>ExpressionBreakThreshold chars), break here
				if len(strings.TrimSpace(current)) > constants.ExpressionBreakThreshold {
					lines = append(lines, strings.TrimSpace(current))
					current = ""
					// Skip whitespace after operator
					for i < len(expression) && (expression[i] == ' ' || expression[i] == '\t') {
						i++
					}
					continue
				}
				continue
			}
		}

		current += string(char)
		i++
	}

	// Add the remaining part
	if strings.TrimSpace(current) != "" {
		lines = append(lines, strings.TrimSpace(current))
	}

	// If we still have very long lines, try to break at parentheses
	var finalLines []string
	for _, line := range lines {
		if len(line) > constants.MaxExpressionLineLength {
			subLines := BreakAtParentheses(line)
			finalLines = append(finalLines, subLines...)
		} else {
			finalLines = append(finalLines, line)
		}
	}

	return finalLines
}

// BreakAtParentheses attempts to break long lines at parentheses for function calls
func BreakAtParentheses(expression string) []string {
	if len(expression) <= constants.MaxExpressionLineLength {
		return []string{expression}
	}

	var lines []string
	current := ""
	parenDepth := 0

	for i := 0; i < len(expression); i++ {
		char := expression[i]
		current += string(char)

		switch char {
		case '(':
			parenDepth++
		case ')':
			parenDepth--

			// If we're back to zero depth and the line is getting long, consider a break
			if parenDepth == 0 && len(current) > 80 && i < len(expression)-1 {
				// Look ahead to see if there's a logical operator
				j := i + 1
				for j < len(expression) && (expression[j] == ' ' || expression[j] == '\t') {
					j++
				}

				if j+1 < len(expression) && (expression[j:j+2] == "||" || expression[j:j+2] == "&&") {
					// Add the operator to current line and break
					for k := i + 1; k < j+2; k++ {
						current += string(expression[k])
					}
					lines = append(lines, strings.TrimSpace(current))
					current = ""
					i = j + 2 - 1 // Set to j+2-1 so the loop increment makes i = j+2

					// Skip whitespace after operator
					for i+1 < len(expression) && (expression[i+1] == ' ' || expression[i+1] == '\t') {
						i++
					}
				}
			}
		}
	}

	// Add remaining part
	if strings.TrimSpace(current) != "" {
		lines = append(lines, strings.TrimSpace(current))
	}

	return lines
}

// NormalizeExpressionForComparison normalizes an expression by removing extra spaces and newlines
// This is used for comparing multiline expressions with their single-line equivalents
func NormalizeExpressionForComparison(expression string) string {
	// Replace newlines and tabs with spaces
	normalized := strings.ReplaceAll(expression, "\n", " ")
	normalized = strings.ReplaceAll(normalized, "\t", " ")

	// Replace multiple spaces with single spaces
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}

	// Trim leading and trailing spaces
	return strings.TrimSpace(normalized)
}
