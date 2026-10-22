package aw

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
)

// ExecutionBlock is an authored AW execution scope, not a compiled Actions job.
// Fields and step nodes are read-only views of the parsed document.
type ExecutionBlock struct {
	Path   string
	Fields map[string]any
	Steps  []ExecutionStep
}

type ExecutionStep struct {
	Path   string
	Fields map[string]any
	Post   bool
}

func (block ExecutionBlock) FieldPath(field string) string {
	if block.Path == "" {
		return field
	}
	return block.Path + "." + field
}

// ExecutionBlocks visits only explicit execution nodes. Imports, generated
// steps, engine configuration and compiler defaults are not reconstructed.
func ExecutionBlocks(ctx context.Context, document *Document) ([]ExecutionBlock, error) {
	if document == nil {
		return nil, errors.New("AW execution traversal requires a document")
	}
	root, err := executionBlock(ctx, "", document.Frontmatter)
	blocks := []ExecutionBlock{root}
	failures := []error{err}
	if outputs, ok := document.Frontmatter["safe-outputs"].(map[string]any); ok {
		if outputs["steps"] != nil || outputs["post-steps"] != nil {
			block, err := executionBlock(ctx, "safe-outputs", outputs)
			blocks = append(blocks, block)
			failures = append(failures, err)
		}
	}
	for _, section := range []string{"jobs", "safe-jobs", "safe-outputs.jobs"} {
		if err := ctx.Err(); err != nil {
			return blocks, errors.Join(append(failures, err)...)
		}
		raw := document.Frontmatter[section]
		if section == "safe-outputs.jobs" {
			raw = nil
			if outputs, ok := document.Frontmatter["safe-outputs"].(map[string]any); ok {
				raw = outputs["jobs"]
			}
		}
		if raw == nil {
			continue
		}
		jobs, ok := raw.(map[string]any)
		if !ok {
			failures = append(failures, fmt.Errorf("AW execution traversal requires a mapping at %s", section))
			continue
		}
		for _, id := range slices.Sorted(maps.Keys(jobs)) {
			if err := ctx.Err(); err != nil {
				return blocks, errors.Join(append(failures, err)...)
			}
			path := section + "." + id
			fields, ok := jobs[id].(map[string]any)
			if !ok {
				failures = append(failures, fmt.Errorf("AW execution traversal requires a mapping at %s", path))
				continue
			}
			block, err := executionBlock(ctx, path, fields)
			blocks = append(blocks, block)
			failures = append(failures, err)
		}
	}
	return blocks, errors.Join(failures...)
}

func executionBlock(ctx context.Context, path string, fields map[string]any) (ExecutionBlock, error) {
	block := ExecutionBlock{Path: path, Fields: fields}
	var failures []error
	for _, key := range []string{"steps", "post-steps"} {
		raw := fields[key]
		if raw == nil {
			continue
		}
		fieldPath := key
		if path != "" {
			fieldPath = path + "." + key
		}
		steps, ok := raw.([]any)
		if !ok {
			failures = append(failures, fmt.Errorf("AW execution traversal requires a step list at %s", fieldPath))
			continue
		}
		for index, rawStep := range steps {
			if err := ctx.Err(); err != nil {
				return block, errors.Join(append(failures, err)...)
			}
			stepPath := fmt.Sprintf("%s[%d]", fieldPath, index)
			step, ok := rawStep.(map[string]any)
			if !ok {
				failures = append(failures, fmt.Errorf("AW execution traversal requires a step mapping at %s", stepPath))
				continue
			}
			block.Steps = append(block.Steps, ExecutionStep{Path: stepPath, Fields: step, Post: key == "post-steps"})
		}
	}
	return block, errors.Join(failures...)
}
