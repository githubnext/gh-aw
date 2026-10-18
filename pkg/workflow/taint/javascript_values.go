package taint

import (
	"maps"
	"reflect"
	"slices"
)

type jsValue struct {
	sources value
	ref     string
	text    string
	literal bool
	kind    jsValueKind
	lines   jsLineSafety
	mask    jsMaskState
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
	for i, current := range values {
		result.sources = union(result.sources, current.taint())
		if i == 0 {
			result.kind = current.kind
		} else if result.kind != current.kind {
			result.kind = jsUnknown
		}
	}
	return result
}

func (v jsValue) property(key string) jsValue {
	// Metadata is public, but an explicit array length assignment may itself
	// contain a numeric secret. Object property names alone prove no type.
	if key == "length" && (v.kind == jsString || v.kind == jsBuffer ||
		v.kind == jsArray && len(v.fields["length"].taint()) == 0 && len(v.fields["*"].taint()) == 0) {
		return jsValue{kind: jsNumber}
	}
	if current, ok := v.fields[key]; ok {
		current.sources = union(current.sources, v.sources)
		if current.ref == "" && v.ref != "" {
			current.ref = v.ref + "." + key
		}
		if key != "*" {
			current.sources = union(current.sources, v.fields["*"].taint())
		}
		return current
	}
	result := jsValue{sources: union(v.sources, v.fields["*"].taint())}
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
		j.bindPattern(node, assigned)
	case "PropertyAccessExpression", "ElementAccessExpression":
		base := j.eval(node.Expression)
		key := jsName(node.Name)
		if node.Kind == "ElementAccessExpression" {
			key, _ = j.key(node.ArgumentExpression)
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

func (j *javascript) bindPattern(node *JavaScriptNode, assigned jsValue) {
	excluded := make(map[string]struct{})
	for i, child := range node.Elements {
		key := jsIndex(i)
		rest := jsHasKind(child.Other, "DotDotDotToken")
		if node.Kind == "ObjectBindingPattern" {
			if rest {
				v := jsValue{sources: assigned.sources, fields: make(map[string]jsValue)}
				// Object rest copies own fields, not their union. Nested
				// objects still alias the original, as JavaScript requires.
				maps.Copy(v.fields, assigned.fields)
				for key := range excluded {
					delete(v.fields, key)
				}
				j.bind(child.Name, v)
				continue
			}
			key = jsName(child.PropertyName)
			if child.PropertyName != nil && child.PropertyName.Kind == "ComputedPropertyName" {
				var known bool
				key, known = j.key(child.PropertyName.Expression)
				if !known {
					j.bind(child.Name, j.eval(child.Initializer))
					continue
				}
			} else if key == "" {
				key = jsName(child.Name)
			}
			excluded[key] = struct{}{}
		}
		if rest {
			key = "*"
		}
		v := jsUnionPreserving(assigned.property(key), j.eval(child.Initializer))
		j.bind(child.Name, v)
	}
}

func jsUnionPreserving(a, b jsValue) jsValue {
	return jsMerge(a, b, make(map[[2]uintptr]bool))
}

func jsMerge(a, b jsValue, seen map[[2]uintptr]bool) jsValue {
	if b.kind == jsUnknown && b.ref == "" && b.fn == nil && b.fields == nil && !b.literal && len(b.sources) == 0 {
		return a
	}
	if a.kind == jsUnknown && a.ref == "" && a.fn == nil && a.fields == nil && !a.literal && len(a.sources) == 0 {
		return b
	}
	a.sources = union(a.sources, b.sources)
	if a.kind != b.kind {
		a.kind = jsUnknown
	}
	a.lines &= b.lines
	if a.mask != b.mask {
		a.mask = jsNotMask
	}
	pair := [2]uintptr{reflect.ValueOf(a.fields).Pointer(), reflect.ValueOf(b.fields).Pointer()}
	if pair[0] != 0 || pair[1] != 0 {
		if pair[0] != 0 && pair[0] == pair[1] || seen[pair] {
			return a
		}
		seen[pair] = true
	}
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
