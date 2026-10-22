package lint

import (
	"context"
	"errors"
)

// ArtipackedRule checks credential persistence, not artifact contents or whether
// a later git operation legitimately needs the checkout credential.
func ArtipackedRule() Rule {
	return Rule{
		ID: "artipacked", Description: "Avoid persisting checkout credentials in later execution steps",
		Severity: Warning, Targets: ActionsAST | AWAST, Check: checkArtipacked,
	}
}

func checkArtipacked(ctx context.Context, input Input) ([]Diagnostic, error) {
	blocks, err := executionPolicyBlocks(ctx, input)
	failures := []error{err}
	var diagnostics []Diagnostic
	for _, block := range blocks {
		for _, step := range block.Steps {
			if err := ctx.Err(); err != nil {
				return diagnostics, errors.Join(append(failures, err)...)
			}
			if actionName(step.Fields["uses"]) != "actions/checkout" {
				continue
			}
			with, err := policyMapping(step.Fields["with"], step.Path+".with")
			if err != nil {
				failures = append(failures, err)
				continue
			}
			path := step.Path + ".with.persist-credentials"
			enabled, known, err := actionInputBoolean(ctx, with["persist-credentials"], path)
			if err != nil {
				failures = append(failures, err)
				continue
			}
			if known && !enabled {
				continue
			}
			if _, explicit := with["persist-credentials"]; !explicit {
				path = step.Path + ".uses"
			}
			diagnostics = append(diagnostics, Diagnostic{
				Path:    path,
				Message: "Checkout may persist credentials for later steps or artifact uploads; set persist-credentials to false unless later authenticated git operations require it",
			})
		}
	}
	return diagnostics, errors.Join(failures...)
}
