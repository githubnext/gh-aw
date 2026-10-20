package lint

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/github/gh-aw/pkg/workflow/taint"
	"github.com/rhysd/actionlint"
)

func executionPolicyBlocks(ctx context.Context, input Input) ([]aw.ExecutionBlock, error) {
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	if input.Workflow == nil {
		return aw.ExecutionBlocks(ctx, input.Document)
	}
	var blocks []aw.ExecutionBlock
	var failures []error
	for _, job := range input.Workflow.Jobs {
		block := aw.ExecutionBlock{Path: "jobs." + string(job.ID)}
		for index, step := range job.Steps {
			if err := ctx.Err(); err != nil {
				return blocks, errors.Join(append(failures, err)...)
			}
			path := fmt.Sprintf("%s.steps[%d]", block.Path, index)
			fields, err := actions.LowerStep(step, job.ID, input.Workflow)
			if err != nil {
				failures = append(failures, fmt.Errorf("cannot inspect execution policy at %s: invalid Actions step; correct the step before linting", path))
				continue
			}
			block.Steps = append(block.Steps, aw.ExecutionStep{Path: path, Fields: fields})
		}
		blocks = append(blocks, block)
	}
	return blocks, errors.Join(failures...)
}

func actionName(value any) string {
	uses, ok := value.(string)
	if !ok {
		return ""
	}
	name, ref, found := strings.Cut(uses, "@")
	if !found || ref == "" || strings.Contains(uses, "${{") {
		return ""
	}
	return strings.ToLower(name)
}

// Action inputs accept both YAML booleans and their string forms. Only literal
// expressions are evaluated; a dynamic flag is not proof of disablement.
func actionInputBoolean(ctx context.Context, value any, path string) (bool, bool, error) {
	switch value := value.(type) {
	case bool:
		return value, true, nil
	case string:
		if strings.EqualFold(value, "true") || strings.EqualFold(value, "false") {
			return strings.EqualFold(value, "true"), true, nil
		}
		text := strings.TrimSpace(value)
		enabled, known := false, false
		err := taint.VisitInterpolations(ctx, text, path, func(start, end, _ int, node actionlint.ExprNode) error {
			if start == 0 && end == len(text) {
				switch node := node.(type) {
				case *actionlint.BoolNode:
					enabled, known = node.Value, true
				case *actionlint.StringNode:
					enabled = strings.EqualFold(node.Value, "true")
					known = enabled || strings.EqualFold(node.Value, "false")
				}
			}
			return ctx.Err()
		})
		return enabled, known, err
	case nil:
		return false, false, nil
	default:
		return false, false, fmt.Errorf("action policy requires a boolean or string input at %s; correct the action inputs before linting", path)
	}
}
