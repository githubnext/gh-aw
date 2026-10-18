package lint

import (
	"context"
	"errors"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

func PullRequestTargetRolesRule() Rule {
	return Rule{
		ID: "pull-request-target-roles", Description: "Restrict privileged pull_request_target admission to trusted repository roles",
		ThreatRuleID: "CTR-008",
		Severity:     Warning, Targets: AWAST, Check: checkPullRequestTargetRoles,
	}
}

func checkPullRequestTargetRoles(ctx context.Context, input Input) ([]Diagnostic, error) {
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
	if _, triggered := on["pull_request_target"]; !triggered {
		return nil, nil
	}
	raw, specified := on["roles"]
	if !specified {
		return nil, nil
	}
	var roles []string
	var err error
	if role, scalar := raw.(string); scalar {
		roles = []string{role}
	} else {
		roles, err = policyStrings(ctx, raw, "on.roles")
	}
	unsafe := false
	for _, role := range roles {
		if ctx.Err() != nil {
			return nil, errors.Join(err, ctx.Err())
		}
		unsafe = unsafe || role == "read" || role == "all"
	}
	if !unsafe {
		return nil, err
	}
	severity, severityErr := sourcePolicySeverity(input, "pull-request-target-roles")
	if severityErr != nil {
		return nil, errors.Join(err, severityErr)
	}
	if severity == Warning {
		features, featureErr := policyMapping(input.Document.Frontmatter["features"], "features")
		err = errors.Join(err, featureErr)
		flag := string(constants.DangerouslyAllowRolesReadAndPullRequestTargetFeatureFlag)
		if value, exact := features[flag]; exact {
			if value == true {
				return nil, err
			}
		} else {
			for key, value := range features {
				if strings.EqualFold(key, flag) && value == true {
					return nil, err
				}
			}
		}
	}
	return []Diagnostic{{Severity: severity, Path: "on.roles",
		Message: "pull_request_target admits untrusted contributors through read or all roles with access to secrets; use pull_request or restrict on.roles to trusted repository roles. Checkout disablement and trigger-risk acknowledgment do not authorize this combination",
	}}, err
}
