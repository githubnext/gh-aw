package taint

import (
	"reflect"
	"slices"
)

type jsValue struct {
	sources value
	ref     string
	text    string
	literal bool
	fields  map[string]jsValue
	fn      *jsFunction
}

type jsFunction struct {
	node *JavaScriptNode
	env  map[string]jsValue
	file string
}

func (v jsValue) taint() value {
	return v.collect(make(map[uintptr]bool))
}

func (v jsValue) collect(seen map[uintptr]bool) value {
	result := v.sources
	id := reflect.ValueOf(v.fields).Pointer()
	if id == 0 || seen[id] {
		return result
	}
	seen[id] = true
	for _, child := range v.fields {
		result = union(result, child.collect(seen))
	}
	return result
}

func jsUnion(values ...jsValue) jsValue {
	result := jsValue{}
	for _, current := range values {
		result.sources = union(result.sources, current.taint())
	}
	return result
}

func (v jsValue) property(key string) jsValue {
	if current, ok := v.fields[key]; ok {
		current.sources = union(current.sources, v.sources)
		return current
	}
	result := jsValue{sources: v.sources}
	if key == "*" {
		result.sources = v.taint()
	}
	if v.ref != "" {
		result.ref = v.ref + "." + key
	}
	return result
}

func (j *javascript) bind(node *JavaScriptNode, assigned jsValue) {
	if node == nil {
		return
	}
	switch node.Kind {
	case "Identifier":
		old := j.env[node.Text]
		j.env[node.Text] = jsUnionPreserving(old, assigned)
	case "ObjectBindingPattern", "ArrayBindingPattern":
		for i, child := range node.Elements {
			key := jsIndex(i)
			if node.Kind == "ObjectBindingPattern" {
				key = jsName(child.PropertyName)
				if key == "" {
					key = jsName(child.Name)
				}
			}
			// Rest bindings may select any member, not just their local name.
			if jsHasKind(child.Other, "DotDotDotToken") {
				key = "*"
			}
			v := assigned.property(key)
			v = jsUnionPreserving(v, j.eval(child.Initializer))
			j.bind(child.Name, v)
		}
	case "PropertyAccessExpression", "ElementAccessExpression":
		base := j.eval(node.Expression)
		key := jsName(node.Name)
		if node.Kind == "ElementAccessExpression" {
			key = j.key(node.ArgumentExpression)
		}
		created := base.fields == nil
		if base.fields == nil {
			base.fields = make(map[string]jsValue)
		}
		base.fields[key] = jsUnionPreserving(base.fields[key], assigned)
		if created {
			j.bind(node.Expression, base)
		}
	default:
		j.gap(node, "JavaScript assignment target is not modeled")
	}
}

func jsUnionPreserving(a, b jsValue) jsValue {
	return jsMerge(a, b, make(map[[2]uintptr]bool))
}

func jsMerge(a, b jsValue, seen map[[2]uintptr]bool) jsValue {
	if b.ref == "" && b.fn == nil && len(b.fields) == 0 && !b.literal && len(b.sources) == 0 {
		return a
	}
	if a.ref == "" && a.fn == nil && len(a.fields) == 0 && !a.literal && len(a.sources) == 0 {
		return b
	}
	a.sources = union(a.sources, b.sources)
	pair := [2]uintptr{reflect.ValueOf(a.fields).Pointer(), reflect.ValueOf(b.fields).Pointer()}
	if pair[0] != 0 && pair[0] == pair[1] || seen[pair] {
		return a
	}
	seen[pair] = true
	if a.fields != nil || b.fields != nil {
		if a.fields == nil {
			a.fields = b.fields
		} else {
			for key, v := range b.fields {
				a.fields[key] = jsMerge(a.fields[key], v, seen)
			}
		}
	}
	if a.ref != b.ref || a.fn != b.fn || a.literal != b.literal || a.text != b.text {
		a.sources = union(a.sources, b.taint())
		a.ref, a.fn, a.literal = "", nil, false
	}
	return a
}

func jsName(n *JavaScriptNode) string {
	if n == nil {
		return ""
	}
	return n.Text
}

func jsHasKind(nodes []*JavaScriptNode, kind string) bool {
	return slices.ContainsFunc(nodes, func(n *JavaScriptNode) bool { return n.Kind == kind })
}
