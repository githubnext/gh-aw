package actions

import (
	"errors"
	"fmt"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
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
	replacement, err := sourceScalarReplacement(source, replacement)
	if err != nil {
		return nil, err
	}
	comment := source.GetComment()
	if alias, ok := source.(*ast.AliasNode); ok && comment == nil {
		comment = alias.Value.GetComment()
	}
	if err := replacement.SetComment(comment); err != nil {
		return nil, fmt.Errorf("preserving workflow value comment: %w", err)
	}
	shiftSourceNode(replacement, source.GetToken().Position.Column-replacement.GetToken().Position.Column)
	return replacement, nil
}

func sourceMappingRenames(before, after map[string]*ast.MappingValueNode) map[string]string {
	renames := make(map[string]string)
	for oldKey, oldField := range before {
		if after[oldKey] != nil {
			continue
		}
		var candidate string
		candidates := 0
		for newKey, newField := range after {
			if before[newKey] == nil && sameSourceValue(oldField.Value, newField.Value) {
				candidate = newKey
				candidates++
			}
		}
		if candidates != 1 {
			continue
		}
		matches := 0
		for key, field := range before {
			if after[key] == nil && sameSourceValue(field.Value, oldField.Value) {
				matches++
			}
		}
		if matches == 1 {
			renames[oldKey] = candidate
		}
	}
	return renames
}

func renameSourceField(field, replacement *ast.MappingValueNode, newKey string) bool {
	key, ok := field.Key.(*ast.StringNode)
	if !ok || replacement == nil {
		return false
	}
	// Reuse an unambiguous rename's syntax so its value and comments
	// remain attached, including inside sequence mappings.
	key.Value, key.Token.Value = newKey, newKey
	if key.Token.Type != token.SingleQuoteType && key.Token.Type != token.DoubleQuoteType {
		key.Token.Type = replacement.Key.GetToken().Type
	}
	return true
}

func mergeSourceExistingFields(entries []*ast.MappingValueNode, original, before, after map[string]*ast.MappingValueNode) ([]*ast.MappingValueNode, error) {
	renames := sourceMappingRenames(before, after)
	var fields []*ast.MappingValueNode
	for _, field := range entries {
		key := sourceKey(field.Key)
		oldField, newField := before[key], after[key]
		if oldField != nil && newField == nil {
			newKey, renamed := renames[key]
			if !renamed || original[newKey] != nil || !renameSourceField(field, after[newKey], newKey) {
				continue
			}
			newField = after[newKey]
			original[newKey] = field
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
			shiftSourceNode(value, field.Value.GetToken().Position.Column-value.GetToken().Position.Column)
			field.Value = value
		}
		fields = append(fields, field)
	}
	return fields, nil
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
	fields, err := mergeSourceExistingFields(entries, original, oldFields, newFields)
	if err != nil {
		return nil, err
	}
	for _, field := range sourceMappingEntries(after) {
		key := sourceKey(field.Key)
		if original[key] != nil || (oldFields[key] != nil && sameSourceValue(oldFields[key].Value, field.Value)) {
			continue
		}
		shiftSourceNode(field, column-field.Key.GetToken().Position.Column)
		if mapping, ok := source.(*ast.MappingNode); ok && mapping.IsFlowStyle {
			field.SetIsFlowStyle(true)
		}
		fields = append(fields, field)
	}
	if mapping, ok := source.(*ast.MappingNode); ok {
		mapping.Values = fields
		if mapping.IsFlowStyle {
			return flowSourceNode(mapping)
		}
		return mapping, nil
	}
	return ast.Mapping(source.GetToken(), false, fields...), nil
}
