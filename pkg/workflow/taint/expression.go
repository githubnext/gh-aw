package taint

import (
	"fmt"
	"strings"

	"github.com/rhysd/actionlint"
)

// interpolate uses the Actions lexer, including its quoted-string handling, so
// literal '}}' and doubled single quotes cannot truncate an expression.
func (a *analyzer) interpolate(text, path string, bindings scope, replace func(int, value) string) (string, value, error) {
	var rendered strings.Builder
	var sources value
	count := 0
	for {
		start := strings.Index(text, "${{")
		if start < 0 {
			rendered.WriteString(text)
			return rendered.String(), sources, nil
		}
		rendered.WriteString(text[:start])
		body := text[start+3:]
		lexer := actionlint.NewExprLexer(body)
		node, err := actionlint.NewExprParser().Parse(lexer)
		if err != nil || lexer.Offset() < 2 || lexer.Offset() > len(body) {
			return "", nil, fmt.Errorf("invalid Actions expression at %s", path)
		}
		end := lexer.Offset()
		if body[end-2:end] != "}}" {
			return "", nil, fmt.Errorf("unterminated Actions expression at %s", path)
		}
		current := a.expression(node, path, bindings)
		sources = union(sources, current)
		if replace != nil {
			rendered.WriteString(replace(count, current))
		} else {
			rendered.WriteString(text[start : start+3+end])
		}
		text = body[end:]
		count++
	}
}

func (a *analyzer) text(text, path string, bindings scope) (value, error) {
	_, sources, err := a.interpolate(text, path, bindings, nil)
	return sources, err
}

func (a *analyzer) expression(node actionlint.ExprNode, path string, bindings scope) value {
	if a.rule.IsFilter != nil && a.rule.IsFilter(node) {
		return nil
	}
	if ref, ok := reference(node); ok {
		result := a.source(ref, path, bindings)
		// Computed indexes may contain sensitive data even below a dereference.
		actionlint.VisitExprNode(node, func(child, _ actionlint.ExprNode, entering bool) {
			if index, ok := child.(*actionlint.IndexAccessNode); ok && entering {
				result = union(result, a.expression(index.Index, path, bindings))
			}
		})
		return result
	}
	var result value
	switch n := node.(type) {
	case *actionlint.FuncCallNode:
		for _, arg := range n.Args {
			result = union(result, a.expression(arg, path, bindings))
		}
	case *actionlint.LogicalOpNode:
		result = union(a.expression(n.Left, path, bindings), a.expression(n.Right, path, bindings))
	case *actionlint.CompareOpNode:
		result = union(a.expression(n.Left, path, bindings), a.expression(n.Right, path, bindings))
	case *actionlint.NotOpNode:
		result = a.expression(n.Operand, path, bindings)
	case *actionlint.ObjectDerefNode:
		result = a.expression(n.Receiver, path, bindings)
	case *actionlint.IndexAccessNode:
		result = union(a.expression(n.Operand, path, bindings), a.expression(n.Index, path, bindings))
	case *actionlint.ArrayDerefNode:
		result = a.expression(n.Receiver, path, bindings)
	}
	return result
}

func reference(node actionlint.ExprNode) (string, bool) {
	switch n := node.(type) {
	case *actionlint.VariableNode:
		return strings.ToLower(n.Name), true
	case *actionlint.ObjectDerefNode:
		base, ok := reference(n.Receiver)
		return base + "." + strings.ToLower(n.Property), ok
	case *actionlint.IndexAccessNode:
		base, ok := reference(n.Operand)
		key := "*"
		if literal, literalIndex := n.Index.(*actionlint.StringNode); literalIndex {
			key = strings.ToLower(literal.Value)
		}
		return base + "." + key, ok
	case *actionlint.ArrayDerefNode:
		base, ok := reference(n.Receiver)
		return base + ".*", ok
	default:
		return "", false
	}
}
