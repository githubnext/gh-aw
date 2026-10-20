package taint

import (
	"context"
	"strings"

	"github.com/rhysd/actionlint"
)

func (a *analyzer) interpolate(text, path string, bindings scope, replace func(int, value) string) (string, value, error) {
	var rendered strings.Builder
	var sources value
	count, offset := 0, 0
	err := VisitInterpolations(context.Background(), text, path, func(start, end, _ int, node actionlint.ExprNode) error {
		rendered.WriteString(text[offset:start])
		current := a.expression(node, path, bindings)
		sources = union(sources, current)
		if replace != nil {
			rendered.WriteString(replace(count, current))
		} else {
			rendered.WriteString(text[start:end])
		}
		offset = end
		count++
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	rendered.WriteString(text[offset:])
	return rendered.String(), sources, nil
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
