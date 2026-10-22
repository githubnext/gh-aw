package lint

import (
	"context"
	"fmt"
	"strings"
)

// WorkflowRunScopeRule checks the observable trigger-selection portion of
// CTR-021, not workflow provenance or runtime authorization. Paths address the
// Actions AST; Diagnostic.Line remains reserved for script-relative locations.
//
// AW sources default to strict unless strict is explicitly false, matching the
// compiler's source policy. Plain Actions input has no AW strict setting and
// receives warnings for branch scope. CLI/repository strict overrides are not
// represented by Input and must be enforced by the caller.
//
// Branch-list checks strengthen the compiler's current branches-key presence
// check. They require positive scope rather than an exclusion-only policy; they
// do not prove the full glob language or which upstream branches are trusted.
func WorkflowRunScopeRule() Rule {
	return Rule{
		ID:           "workflow-run-scope",
		ThreatRuleID: "CTR-021",
		Description:  "Require workflow selection and positive branch scope for workflow_run triggers",
		Severity:     Error,
		Targets:      ActionsAST | AWAST,
		Check:        checkWorkflowRunScope,
	}
}

func checkWorkflowRunScope(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	var on any
	if input.Workflow != nil {
		on = input.Workflow.On
	} else {
		on = input.Document.Frontmatter["on"]
	}
	config, path, found, err := workflowRunTrigger(ctx, on)
	if err != nil || !found {
		return nil, err
	}
	fields, ok := config.(map[string]any)
	if !ok {
		return []Diagnostic{{
			Severity: Error, Path: path,
			Message: "workflow_run requires configuration under an on mapping with a non-empty workflows selection; configure workflows and branches explicitly",
		}}, nil
	}
	_, valid, err := workflowRunStrings(ctx, fields["workflows"], true)
	if err != nil {
		return nil, err
	}
	if !valid {
		return []Diagnostic{{
			Severity: Error, Path: path + ".workflows",
			Message: "workflow_run requires a non-empty workflows selection containing only non-empty workflow names; name at least one upstream workflow",
		}}, nil
	}

	branches, hasBranches := fields["branches"]
	_, hasIgnore := fields["branches-ignore"]
	if hasBranches && hasIgnore {
		return []Diagnostic{{
			Severity: Error, Path: path + ".branches-ignore",
			Message: "workflow_run cannot combine branches and branches-ignore; use branches with optional negative patterns instead",
		}}, nil
	}
	scoped, err := workflowRunBranchScope(ctx, branches)
	if err != nil {
		return nil, err
	}
	if scoped {
		return nil, nil
	}
	severity := Warning
	if input.Document != nil {
		if strict, ok := input.Document.Frontmatter["strict"].(bool); !ok || strict {
			severity = Error
		}
	}
	message := "workflow_run lacks positive branch scope; set branches to a non-empty list of specific branch names or scoped patterns, not wildcard-only or negative-only filters"
	if hasIgnore {
		message = "workflow_run branches-ignore does not establish positive branch scope; replace it with branches naming trusted branches or scoped patterns"
	}
	return []Diagnostic{{Severity: severity, Path: path + ".branches", Message: message}}, nil
}

func workflowRunBranchScope(ctx context.Context, branches any) (bool, error) {
	patterns, valid, err := workflowRunStrings(ctx, branches, false)
	if err != nil || !valid {
		return false, err
	}
	var active string
	hasPositive, broad := false, false
	for _, pattern := range patterns {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		pattern = strings.TrimSpace(pattern)
		if excluded, negative := strings.CutPrefix(pattern, "!"); negative {
			// GitHub applies filters in order. A universal exclusion clears earlier
			// positive scope before later allowlist entries can re-include it.
			if strings.Trim(excluded, "*?/+") == "" {
				active = ""
				broad = false
				continue
			}
			if active != "" && strings.TrimSpace(excluded) == strings.TrimSpace(active) {
				active = ""
			}
			continue
		}
		hasPositive = true
		active = pattern
		// A broad positive pattern defeats narrower entries in the same allowlist,
		// unless a later universal exclusion clears the whole set.
		if strings.Trim(pattern, "*?/+") == "" {
			broad = true
		}
	}
	return hasPositive && !broad && active != "", nil
}

func workflowRunTrigger(ctx context.Context, on any) (any, string, bool, error) {
	switch triggers := on.(type) {
	case map[string]any:
		config, found := triggers["workflow_run"]
		return config, "on.workflow_run", found, nil
	case string:
		return nil, "on", triggers == "workflow_run", nil
	case []string:
		for index, trigger := range triggers {
			if err := ctx.Err(); err != nil {
				return nil, "", false, err
			}
			if trigger == "workflow_run" {
				return nil, fmt.Sprintf("on[%d]", index), true, nil
			}
		}
	case []any:
		for index, trigger := range triggers {
			if err := ctx.Err(); err != nil {
				return nil, "", false, err
			}
			if name, ok := trigger.(string); ok && name == "workflow_run" {
				return nil, fmt.Sprintf("on[%d]", index), true, nil
			}
			if fields, ok := trigger.(map[string]any); ok {
				if _, found := fields["workflow_run"]; found {
					return nil, fmt.Sprintf("on[%d].workflow_run", index), true, nil
				}
			}
		}
	}
	return nil, "", false, nil
}

// The compiler accepts a scalar workflows name as well as lists. Branch filters
// are lists in the workflow schema. Unlike the compiler's presence check, every
// entry must be non-empty and well-typed to establish a valid selection.
func workflowRunStrings(ctx context.Context, value any, allowScalar bool) ([]string, bool, error) {
	var values []string
	switch entries := value.(type) {
	case string:
		if !allowScalar {
			return nil, false, nil
		}
		values = []string{entries}
	case []string:
		values = entries
	case []any:
		values = make([]string, 0, len(entries))
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return nil, false, err
			}
			text, ok := entry.(string)
			if !ok {
				return nil, false, nil
			}
			values = append(values, text)
		}
	default:
		return nil, false, nil
	}
	if len(values) == 0 {
		return nil, false, nil
	}
	for _, entry := range values {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		if strings.TrimSpace(entry) == "" || strings.Contains(entry, "${{") {
			return nil, false, nil
		}
	}
	return values, true, nil
}
