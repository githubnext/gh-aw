package lint

import (
	"context"
	"errors"
	"slices"
)

func ForkAssociationRule() Rule {
	return Rule{
		ID: "fork-association", Description: "Validate explicit pull-request fork author-association admission",
		ThreatRuleID: "CTR-034",
		Severity:     Error, Targets: AWAST, Check: checkForkAssociations,
	}
}

func checkForkAssociations(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	if input.Document == nil {
		return nil, nil
	}
	on, ok := input.Document.Frontmatter["on"].(map[string]any)
	if !ok {
		return nil, nil
	}
	pr, err := policyMapping(on["pull_request"], "on.pull_request")
	if err != nil {
		return nil, err
	}
	forks, ok := pr["forks"].(map[string]any)
	if !ok {
		// Repository patterns and omitted fork admission do not use associations.
		return nil, nil
	}
	const path = "on.pull_request.forks.author-association"
	raw, specified := forks["author-association"]
	if !specified || raw == nil {
		return []Diagnostic{{Path: path, Message: "Fork association admission requires an explicit author-association list; an empty list admits no forks"}}, nil
	}
	associations, err := policyStrings(ctx, raw, path)
	var diagnostics []Diagnostic
	allowed := []string{"OWNER", "MEMBER", "COLLABORATOR", "CONTRIBUTOR", "FIRST_TIMER", "FIRST_TIME_CONTRIBUTOR", "MANNEQUIN", "NONE"}
	for _, association := range associations {
		if ctx.Err() != nil {
			return diagnostics, errors.Join(err, ctx.Err())
		}
		if !slices.Contains(allowed, association) {
			diagnostics = append(diagnostics, Diagnostic{Path: path,
				Message: "Fork author associations must use GitHub's uppercase author-association enum; remove or correct the invalid entry"})
		}
	}
	return diagnostics, err
}
