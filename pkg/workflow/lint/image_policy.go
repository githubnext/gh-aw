package lint

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/aw"
)

func ImageArgumentRule() Rule {
	return Rule{
		ID: "image-argument", Description: "Reject option-prefixed container image operands",
		ThreatRuleID: "CTR-013",
		Severity:     Error, Targets: ActionsAST | AWAST, Check: checkImageArguments,
	}
}

func checkImageArguments(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	var diagnostics []Diagnostic
	var failures []error
	check := func(raw any, path string) {
		found, err := imageOperand(raw, path)
		diagnostics = append(diagnostics, found...)
		failures = append(failures, err)
	}
	checkServices := func(services map[string]any, path string) error {
		for _, name := range slices.Sorted(maps.Keys(services)) {
			if err := ctx.Err(); err != nil {
				return err
			}
			check(services[name], path+"."+name)
		}
		return nil
	}
	if input.Workflow != nil {
		for _, job := range input.Workflow.Jobs {
			if err := ctx.Err(); err != nil {
				return diagnostics, errors.Join(append(failures, err)...)
			}
			path := "jobs." + string(job.ID)
			check(job.Container, path+".container")
			failures = append(failures, checkServices(job.Services, path+".services"))
		}
	} else {
		blocks, err := aw.ExecutionBlocks(ctx, input.Document)
		failures = append(failures, err)
		for _, block := range blocks {
			if err := ctx.Err(); err != nil {
				return diagnostics, errors.Join(append(failures, err)...)
			}
			check(block.Fields["container"], block.FieldPath("container"))
			services, err := policyMapping(block.Fields["services"], block.FieldPath("services"))
			failures = append(failures, err)
			failures = append(failures, checkServices(services, block.FieldPath("services")))
		}
	}
	if input.Document != nil {
		servers, err := authoredPolicyMapping(ctx, input, "mcp-servers")
		failures = append(failures, err)
		for _, name := range slices.Sorted(maps.Keys(servers)) {
			if err := ctx.Err(); err != nil {
				return diagnostics, errors.Join(append(failures, err)...)
			}
			server, err := policyMapping(servers[name], "mcp-servers."+name)
			failures = append(failures, err)
			check(server["container"], "mcp-servers."+name+".container")
		}
	}
	return diagnostics, errors.Join(failures...)
}

func imageOperand(raw any, field string) ([]Diagnostic, error) {
	if raw == nil {
		return nil, nil
	}
	if fields, mapping := raw.(map[string]any); mapping {
		raw = fields["image"]
		field += ".image"
	}
	image, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("image-argument requires an image string at %s", field)
	}
	if strings.HasPrefix(image, "-") {
		return []Diagnostic{{Path: field,
			Message: "Container image operands must not start with a hyphen because launchers can interpret them as command options"}}, nil
	}
	return nil, nil
}
