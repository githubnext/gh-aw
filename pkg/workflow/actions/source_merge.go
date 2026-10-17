package actions

import (
	"errors"
	"fmt"

	"github.com/goccy/go-yaml/ast"
)

func sameSourceValue(left, right ast.Node) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.String() == right.String()
}

func sourceMapping(node ast.Node) map[string]*ast.MappingValueNode {
	result := make(map[string]*ast.MappingValueNode)
	for _, field := range sourceMappingEntries(node) {
		result[sourceKey(field.Key)] = field
	}
	return result
}

func sourceKey(node ast.MapKeyNode) string {
	if text, ok := node.(*ast.StringNode); ok {
		return text.Value
	}
	return node.GetToken().Value
}

func sourceMappingEntries(node ast.Node) []*ast.MappingValueNode {
	switch value := node.(type) {
	case *ast.MappingNode:
		return value.Values
	case *ast.MappingValueNode:
		return []*ast.MappingValueNode{value}
	default:
		return nil
	}
}

func mergeSourceNode(source, before, after ast.Node) (ast.Node, error) {
	if sameSourceValue(before, after) {
		return source, nil
	}
	if anchor, ok := source.(*ast.AnchorNode); ok {
		value, err := mergeSourceNode(anchor.Value, before, after)
		anchor.Value = value
		return anchor, err
	}
	if len(sourceMappingEntries(source)) > 0 && len(sourceMappingEntries(after)) > 0 {
		return mergeSourceMapping(source, before, after)
	}
	if sequence, ok := source.(*ast.SequenceNode); ok {
		oldSequence, oldOK := before.(*ast.SequenceNode)
		newSequence, newOK := after.(*ast.SequenceNode)
		if oldOK && newOK {
			return mergeSourceSequence(sequence, oldSequence, newSequence)
		}
	}
	return replaceSourceNode(source, after)
}

func replaceSourceNode(source, replacement ast.Node) (ast.Node, error) {
	if source == nil || replacement == nil {
		return replacement, nil
	}
	comment := source.GetComment()
	if alias, ok := source.(*ast.AliasNode); ok && comment == nil {
		comment = alias.Value.GetComment()
	}
	if err := replacement.SetComment(comment); err != nil {
		return nil, fmt.Errorf("preserving workflow value comment: %w", err)
	}
	replacement.AddColumn(source.GetToken().Position.Column - replacement.GetToken().Position.Column)
	return replacement, nil
}

func mergeSourceMapping(source, before, after ast.Node) (ast.Node, error) {
	original := sourceMapping(source)
	entries := sourceMappingEntries(source)
	if len(entries) == 0 {
		return nil, errors.New("cannot merge a workflow mapping without source entries")
	}
	column := source.GetToken().Position.Column
	for _, field := range entries {
		column = min(column, field.Key.GetToken().Position.Column)
	}
	oldFields, newFields := sourceMapping(before), sourceMapping(after)
	var fields []*ast.MappingValueNode
	for _, field := range sourceMappingEntries(source) {
		key := sourceKey(field.Key)
		oldField, newField := oldFields[key], newFields[key]
		if oldField != nil && newField == nil {
			continue
		}
		if newField != nil {
			var oldValue ast.Node
			if oldField != nil {
				oldValue = oldField.Value
			}
			value, err := mergeSourceNode(field.Value, oldValue, newField.Value)
			if err != nil {
				return nil, err
			}
			if err := field.Replace(value); err != nil {
				return nil, err
			}
		}
		fields = append(fields, field)
	}
	for _, field := range sourceMappingEntries(after) {
		key := sourceKey(field.Key)
		if original[key] != nil || (oldFields[key] != nil && sameSourceValue(oldFields[key].Value, field.Value)) {
			continue
		}
		field.AddColumn(column - field.Key.GetToken().Position.Column)
		if mapping, ok := source.(*ast.MappingNode); ok && mapping.IsFlowStyle {
			field.SetIsFlowStyle(true)
		}
		fields = append(fields, field)
	}
	if mapping, ok := source.(*ast.MappingNode); ok {
		mapping.Values = fields
		return mapping, nil
	}
	return ast.Mapping(source.GetToken(), false, fields...), nil
}
