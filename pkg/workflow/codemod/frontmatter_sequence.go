package codemod

import (
	"errors"
	"slices"
	"strings"

	yamlv3 "gopkg.in/yaml.v3"
)

// MergeSequence appends unique source entries to a destination sequence, retaining
// authored entry comments when both sequences use block syntax.
func (f *Frontmatter) MergeSequence(path, destination []string) error {
	if slices.Equal(path, destination) {
		return nil
	}
	sourceValue, found := f.Lookup(path...)
	source, ok := sourceValue.([]any)
	if !found || !ok {
		return errors.New("frontmatter sequence merge requires a source sequence")
	}
	targetValue, found := f.Lookup(destination...)
	target, ok := targetValue.([]any)
	if !found || !ok {
		return errors.New("frontmatter sequence merge requires a destination sequence")
	}
	_, sourceNode, err := f.entry(path)
	if err != nil {
		return err
	}
	_, targetNode, err := f.entry(destination)
	if err != nil {
		return err
	}
	var additions []int
	merged := slices.Clone(target)
	for index, value := range source {
		found, err := containsValue(merged, value)
		if err != nil {
			return err
		}
		if !found {
			additions = append(additions, index)
			merged = append(merged, value)
		}
	}
	if len(additions) > 0 {
		if sourceNode.Style&yamlv3.FlowStyle != 0 || targetNode.Style&yamlv3.FlowStyle != 0 {
			if err := f.Set(destination, merged); err != nil {
				return err
			}
		} else {
			if err := f.appendSequenceEntries(sourceNode, targetNode, additions); err != nil {
				return err
			}
		}
	}
	return f.Remove(path...)
}

func (f *Frontmatter) appendSequenceEntries(source, target *yamlv3.Node, additions []int) error {
	end, err := f.nodeEnd(target)
	if err != nil {
		return err
	}
	newline := strings.IndexByte(f.content[end:], '\n')
	if newline < 0 {
		return errors.New("cannot locate destination sequence end")
	}
	insert := end + newline + 1
	var text strings.Builder
	for _, index := range additions {
		if index < 0 || index >= len(source.Content) {
			return errors.New("source sequence does not match its authored syntax")
		}
		node := source.Content[index]
		start, err := f.offset(node.Line, 1)
		if err != nil {
			return err
		}
		end, err := f.nodeEnd(node)
		if err != nil {
			return err
		}
		newline := strings.IndexByte(f.content[end:], '\n')
		if newline < 0 {
			return errors.New("cannot locate source sequence entry end")
		}
		end += newline + 1
		oldIndent := strings.Repeat(" ", source.Column-1)
		newIndent := strings.Repeat(" ", target.Column-1)
		for line := range strings.SplitSeq(strings.TrimSuffix(f.content[start:end], "\n"), "\n") {
			text.WriteString(newIndent + strings.TrimPrefix(line, oldIndent) + "\n")
		}
	}
	return f.replace(insert, insert, text.String())
}
