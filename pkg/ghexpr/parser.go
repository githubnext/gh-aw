package ghexpr

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var parserLog = logger.New("ghexpr:parser")

// StripExpressionWrapper removes the ${{ }} wrapper from s if present, trimming
// inner whitespace.  If s is not wrapped, it is returned unchanged.
func StripExpressionWrapper(expression string) string {
	expr := strings.TrimSpace(expression)
	if strings.HasPrefix(expr, "${{") && strings.HasSuffix(expr, "}}") {
		return strings.TrimSpace(expr[3 : len(expr)-2])
	}
	return expr
}

// ExpressionParser holds parser state for a single parse call.
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

// ParseExpression parses expression (without ${{ }} wrappers) into a ConditionNode tree.
// Supports && (AND), || (OR), ! (NOT), and parentheses for grouping.
//
// Example:
//
//	node, err := ghexpr.ParseExpression("condition1 && (condition2 || !condition3)")
func ParseExpression(expression string) (ConditionNode, error) {
	parserLog.Printf("Parsing expression: %s", expression)

	if strings.TrimSpace(expression) == "" {
		return nil, errors.New("empty expression")
	}

	p := &ExpressionParser{}
	tokens, err := p.tokenize(expression)
	if err != nil {
		parserLog.Printf("Failed to tokenize expression: %v", err)
		return nil, err
	}
	p.tokens = tokens
	p.pos = 0

	result, err := p.parseOrExpression()
	if err != nil {
		parserLog.Printf("Failed to parse expression: %v", err)
		return nil, err
	}

	if p.current().kind != tokenEOF {
		return nil, fmt.Errorf("unexpected token '%s' at position %d", p.current().value, p.current().pos)
	}

	parserLog.Printf("Successfully parsed expression with %d tokens", len(tokens))
	return result, nil
}

// tokenize breaks the expression string into tokens.
func (p *ExpressionParser) tokenize(expression string) ([]token, error) {
	parserLog.Printf("Tokenizing expression of length %d", len(expression))
	var tokens []token
	i := 0

	for i < len(expression) {
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
			tokens = append(tokens, token{tokenNot, "!", i})
			i++
		case expression[i] == '(':
			tokens = append(tokens, token{tokenLeftParen, "(", i})
			i++
		case expression[i] == ')':
			tokens = append(tokens, token{tokenRightParen, ")", i})
			i++
		default:
			start := i
			parenCount := 0

			for i < len(expression) {
				ch := expression[i]

				if ch == '\'' || ch == '"' || ch == '`' {
					quote := ch
					i++
					for i < len(expression) {
						if expression[i] == quote {
							i++
							break
						}
						if expression[i] == '\\' && i+1 < len(expression) {
							i += 2
						} else {
							i++
						}
					}
					continue
				}

				if ch == '(' {
					parenCount++
					i++
					continue
				} else if ch == ')' {
					if parenCount > 0 {
						parenCount--
						i++
						continue
					}
					break
				}

				if parenCount == 0 {
					if i+1 < len(expression) {
						next := expression[i : i+2]
						if next == "&&" || next == "||" {
							break
						}
					}
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

func (p *ExpressionParser) parseOrExpression() (ConditionNode, error) {
	left, err := p.parseAndExpression()
	if err != nil {
		return nil, err
	}
	for p.current().kind == tokenOr {
		p.advance()
		right, err := p.parseAndExpression()
		if err != nil {
			return nil, err
		}
		left = &OrNode{Left: left, Right: right}
	}
	return left, nil
}

func (p *ExpressionParser) parseAndExpression() (ConditionNode, error) {
	left, err := p.parseUnaryExpression()
	if err != nil {
		return nil, err
	}
	for p.current().kind == tokenAnd {
		p.advance()
		right, err := p.parseUnaryExpression()
		if err != nil {
			return nil, err
		}
		left = &AndNode{Left: left, Right: right}
	}
	return left, nil
}

func (p *ExpressionParser) parseUnaryExpression() (ConditionNode, error) {
	if p.current().kind == tokenNot {
		p.advance()
		operand, err := p.parseUnaryExpression()
		if err != nil {
			return nil, err
		}
		return &NotNode{Child: operand}, nil
	}
	return p.parsePrimaryExpression()
}

func (p *ExpressionParser) parsePrimaryExpression() (ConditionNode, error) {
	if parserLog.Enabled() {
		parserLog.Printf("Parsing primary expression at token: %s", p.current().value)
	}
	switch p.current().kind {
	case tokenLeftParen:
		p.advance()
		expr, err := p.parseOrExpression()
		if err != nil {
			return nil, err
		}
		if p.current().kind != tokenRightParen {
			return nil, fmt.Errorf("expected ')' at position %d", p.current().pos)
		}
		p.advance()
		return expr, nil

	case tokenLiteral:
		literal := p.current().value
		p.advance()
		return &ExpressionNode{Expression: literal}, nil

	default:
		return nil, fmt.Errorf("unexpected token '%s' at position %d", p.current().value, p.current().pos)
	}
}

func (p *ExpressionParser) current() token {
	if p.pos >= len(p.tokens) {
		return token{tokenEOF, "", -1}
	}
	return p.tokens[p.pos]
}

func (p *ExpressionParser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// VisitExpressionTree walks a ConditionNode tree and calls visitor for each
// [ExpressionNode] leaf found in the tree.  Walking stops and the error is
// returned immediately if visitor returns a non-nil error.
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
	}
	// ComparisonNode, PropertyAccessNode, FunctionCallNode etc. are opaque leaves.
	return nil
}

// BreakLongExpression breaks a long expression string into multiple lines at
// logical operators (|| and &&) for readability.  Strings inside quote
// characters are never split.
func BreakLongExpression(expression string) []string {
	if len(expression) <= int(constants.MaxExpressionLineLength) {
		return []string{expression}
	}

	parserLog.Printf("Breaking long expression: length=%d", len(expression))

	var lines []string
	var current strings.Builder
	i := 0

	for i < len(expression) {
		char := expression[i]

		if char == '\'' || char == '"' || char == '`' {
			quote := char
			current.WriteByte(char)
			i++
			for i < len(expression) {
				current.WriteByte(expression[i])
				if expression[i] == quote {
					i++
					break
				}
				if expression[i] == '\\' && i+1 < len(expression) {
					i++
					if i < len(expression) {
						current.WriteByte(expression[i])
					}
				}
				i++
			}
			continue
		}

		if i+2 <= len(expression) {
			next2 := expression[i : i+2]
			if next2 == "||" || next2 == "&&" {
				current.WriteString(next2)
				i += 2
				if trimmed := strings.TrimSpace(current.String()); len(trimmed) > int(constants.ExpressionBreakThreshold) {
					lines = append(lines, trimmed)
					current.Reset()
					for i < len(expression) && (expression[i] == ' ' || expression[i] == '\t') {
						i++
					}
					continue
				}
				continue
			}
		}

		current.WriteByte(char)
		i++
	}

	if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
		lines = append(lines, trimmed)
	}

	var finalLines []string
	for _, line := range lines {
		if len(line) > int(constants.MaxExpressionLineLength) {
			finalLines = append(finalLines, BreakAtParentheses(line)...)
		} else {
			finalLines = append(finalLines, line)
		}
	}

	return finalLines
}

// BreakAtParentheses attempts to break long lines at parentheses after function calls.
func BreakAtParentheses(expression string) []string {
	if len(expression) <= int(constants.MaxExpressionLineLength) {
		return []string{expression}
	}

	parserLog.Printf("Breaking expression at parentheses: length=%d", len(expression))

	var lines []string
	var current strings.Builder
	parenDepth := 0

	for i := 0; i < len(expression); i++ {
		char := expression[i]
		current.WriteByte(char)

		switch char {
		case '(':
			parenDepth++
		case ')':
			parenDepth--
			if parenDepth == 0 && current.Len() > 80 && i < len(expression)-1 {
				j := i + 1
				for j < len(expression) && (expression[j] == ' ' || expression[j] == '\t') {
					j++
				}
				if j+1 < len(expression) && (expression[j:j+2] == "||" || expression[j:j+2] == "&&") {
					current.WriteString(expression[i+1 : j+2])
					lines = append(lines, strings.TrimSpace(current.String()))
					current.Reset()
					i = j + 2 - 1
					for i+1 < len(expression) && (expression[i+1] == ' ' || expression[i+1] == '\t') {
						i++
					}
				}
			}
		}
	}

	if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
		lines = append(lines, trimmed)
	}

	return lines
}

// HasNewlineInStringLiteral returns true if s contains an actual newline
// character inside a single-quoted GitHub Actions expression string literal.
// This is used to determine whether the YAML if: value needs special encoding.
func HasNewlineInStringLiteral(s string) bool {
	inString := false
	i := 0
	for i < len(s) {
		ch := s[i]
		if ch == '\'' {
			if inString && i+1 < len(s) && s[i+1] == '\'' {
				i += 2
				continue
			}
			inString = !inString
		} else if ch == '\n' && inString {
			return true
		}
		i++
	}
	return false
}

// EscapeForYAMLDoubleQuoted escapes s so it can be placed inside a YAML
// double-quoted scalar ("...").  YAML double-quoted scalars interpret \n, \r,
// \t, \\ and \" escape sequences, so the corresponding actual characters are
// converted to their two-character representations.
func EscapeForYAMLDoubleQuoted(s string) string {
	var b strings.Builder
	for i := range len(s) {
		switch s[i] {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
