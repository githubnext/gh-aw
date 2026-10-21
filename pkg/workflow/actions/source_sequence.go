package actions

import (
	"fmt"
	"slices"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

type sourceCommentVisitor struct{}

func (sourceCommentVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	if sequence, ok := node.(*ast.SequenceNode); ok {
		retainSequenceEntryComments(sequence)
	}
	return sourceCommentVisitor{}
}

func retainSequenceEntryComments(sequence *ast.SequenceNode) {
	if len(sequence.ValueHeadComments) < len(sequence.Values) {
		sequence.ValueHeadComments = append(sequence.ValueHeadComments,
			make([]*ast.CommentGroupNode, len(sequence.Values)-len(sequence.ValueHeadComments))...)
	}
	if !sequence.IsFlowStyle && sequence.Comment != nil {
		for index, head := range sequence.ValueHeadComments {
			sequence.ValueHeadComments[index] = joinSourceComments(sequence.Comment, head)
			sequence.Comment = nil
			break
		}
	}
	for index, entry := range sequence.Entries {
		if entry.LineComment == nil {
			continue
		}
		if index >= 0 && index < len(sequence.ValueHeadComments) {
			comment := sequence.ValueHeadComments[index]
			if comment == nil {
				sequence.ValueHeadComments[index] = entry.LineComment
				continue
			}
			sequence.ValueHeadComments[index] = joinSourceComments(comment, entry.LineComment)
		}
	}
}

func joinSourceComments(first, second *ast.CommentGroupNode) *ast.CommentGroupNode {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	nodes := slices.Clone(first.Comments)
	for _, line := range second.Comments {
		if !slices.Contains(nodes, line) {
			nodes = append(nodes, line)
		}
	}
	return &ast.CommentGroupNode{BaseNode: &ast.BaseNode{}, Comments: nodes}
}

type sourceSequenceEntry struct {
	node  ast.Node
	value string
}

func decodedSourceValues(nodes []ast.Node) ([]sourceSequenceEntry, error) {
	values := make([]sourceSequenceEntry, 0, len(nodes))
	for _, node := range nodes {
		var value any
		if err := yaml.NodeToValue(node, &value); err != nil {
			return nil, fmt.Errorf("decoding workflow sequence entry: %w", err)
		}
		content, err := yaml.MarshalWithOptions(value, yaml.JSON())
		if err != nil {
			return nil, fmt.Errorf("snapshotting workflow sequence entry: %w", err)
		}
		values = append(values, sourceSequenceEntry{node: node, value: string(content)})
	}
	return values, nil
}

func sourceSequenceMatches(before, after []ast.Node) ([]int, error) {
	oldValues, err := decodedSourceValues(before)
	if err != nil {
		return nil, err
	}
	newValues, err := decodedSourceValues(after)
	if err != nil {
		return nil, err
	}
	used := make(map[int]bool)
	matches := make([]int, 0, len(after))
	for _, next := range newValues {
		index := matchSourceSequenceEntry(oldValues, next, used)
		matches = append(matches, index)
		if index >= 0 {
			used[index] = true
		}
	}
	// Reserve stable matches before positional replacements, so inserting a
	// new step cannot steal a comment from a step that moved further down.
	for position, index := range matches {
		if index < 0 && position < len(before) && !used[position] {
			matches[position] = position
			used[position] = true
		}
	}
	return matches, nil
}

func matchSourceSequenceEntry(before []sourceSequenceEntry, next sourceSequenceEntry, used map[int]bool) int {
	for _, key := range []string{"id", "name"} {
		target := sourceMapping(next.node)[key]
		if target == nil {
			continue
		}
		match, count := -1, 0
		for index, entry := range before {
			candidate := sourceMapping(entry.node)[key]
			if candidate != nil && candidate.Value.GetToken().Value == target.Value.GetToken().Value {
				count++
				if !used[index] {
					match = index
				}
			}
		}
		if count == 1 && match >= 0 {
			return match
		}
	}
	for index, candidate := range before {
		if !used[index] && candidate.value == next.value {
			return index
		}
	}
	return -1
}

func mergeSourceSequence(source, before, after *ast.SequenceNode) (ast.Node, error) {
	originalIndices, err := sourceSequenceMatches(source.Values, before.Values)
	if err != nil {
		return nil, err
	}
	matches, err := sourceSequenceMatches(before.Values, after.Values)
	if err != nil {
		return nil, err
	}
	var values []ast.Node
	var comments []*ast.CommentGroupNode
	for position, next := range after.Values {
		var comment *ast.CommentGroupNode
		value := next
		var index int
		if position >= 0 && position < len(matches) {
			index = matches[position]
		} else {
			return nil, fmt.Errorf("workflow sequence match index %d is out of bounds", position)
		}
		if index >= 0 && index < len(before.Values) && index < len(originalIndices) {
			sourceIndex := originalIndices[index]
			if sourceIndex >= 0 && sourceIndex < len(source.Values) {
				value, err = mergeSourceNode(source.Values[sourceIndex], before.Values[index], value)
				if err != nil {
					return nil, err
				}
				if sourceIndex >= 0 && sourceIndex < len(source.ValueHeadComments) {
					comment = source.ValueHeadComments[sourceIndex]
				}
			}
		}
		column := value.GetToken().Position.Column
		// Block mapping tokens point at the first colon, not the first key.
		// Aligning that colon to the sequence item can shift keys below column one.
		for _, field := range sourceMappingEntries(value) {
			column = min(column, field.Key.GetToken().Position.Column)
		}
		shiftSourceNode(value, source.Start.Position.Column+2-column)
		if source.IsFlowStyle {
			value, err = flowSourceNode(value)
			if err != nil {
				return nil, err
			}
		}
		values = append(values, value)
		comments = append(comments, comment)
	}
	source.Values, source.ValueHeadComments = values, comments
	return source, nil
}
