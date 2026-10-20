package lint

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
)

var permissionNamePattern = regexp.MustCompile(`^[a-z][a-z-]*$`)

// PermissionScopeRule checks explicit declarations, not whether individual
// repository writes are authorized for an agent or a safe-output publisher.
func PermissionScopeRule() Rule {
	return Rule{
		ID: "permission-scope", Description: "Avoid blanket write permissions and invalid OIDC read grants",
		ThreatRuleID: "CTR-001",
		Severity:     Warning, Targets: ActionsAST | AWAST, Check: checkPermissionScopes,
	}
}

// ExcessivePermissionsRule applies only to Actions: AW root permissions scope
// the agent job, not every job in the compiled workflow.
func ExcessivePermissionsRule() Rule {
	return Rule{
		ID: "excessive-permissions", Description: "Scope workflow-wide write permissions to the jobs that need them",
		ThreatRuleID: "CTR-001",
		Severity:     Warning, Targets: ActionsAST, Check: checkExcessivePermissions,
	}
}

func checkExcessivePermissions(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	if input.Workflow == nil {
		return nil, nil
	}
	if _, shorthand := input.Workflow.Permissions.(string); shorthand {
		// permission-scope owns blanket grants and malformed shorthands.
		_, err := permissionScope(ctx, input.Workflow.Permissions, "permissions", Warning)
		return nil, err
	}
	fields, err := policyMapping(input.Workflow.Permissions, "permissions")
	if err != nil {
		return nil, err
	}
	var diagnostics []Diagnostic
	var failures []error
	for _, name := range slices.Sorted(maps.Keys(fields)) {
		if err := ctx.Err(); err != nil {
			return diagnostics, errors.Join(append(failures, err)...)
		}
		level, ok := fields[name].(string)
		if !permissionNamePattern.MatchString(name) || !ok || level != "read" && level != "write" && level != "none" {
			failures = append(failures, errors.New("excessive-permissions requires literal permission names and read, write, or none levels at permissions"))
		} else if level == "write" {
			diagnostics = append(diagnostics, Diagnostic{
				Path:    "permissions." + name,
				Message: "Workflow-wide write permission is inherited by jobs without an explicit permission mapping; move the grant to only the jobs that require it",
			})
		}
	}
	return diagnostics, errors.Join(failures...)
}

func checkPermissionScopes(ctx context.Context, input Input) ([]Diagnostic, error) {
	if err := integrityInput(ctx, input); err != nil {
		return nil, err
	}
	severity, err := sourcePolicySeverity(input, "permission-scope")
	if err != nil {
		return nil, err
	}
	if input.Workflow == nil {
		return checkAuthoredPermissions(ctx, input, severity)
	}
	diagnostics, err := permissionScope(ctx, input.Workflow.Permissions, "permissions", severity)
	failures := []error{err}
	for _, job := range input.Workflow.Jobs {
		if err := ctx.Err(); err != nil {
			return diagnostics, errors.Join(append(failures, err)...)
		}
		found, err := permissionScope(ctx, job.Permissions, "jobs."+string(job.ID)+".permissions", severity)
		diagnostics = append(diagnostics, found...)
		failures = append(failures, err)
	}
	return diagnostics, errors.Join(failures...)
}

func permissionScope(ctx context.Context, value any, path string, severity Severity) ([]Diagnostic, error) {
	if text, ok := value.(string); ok {
		switch text {
		case "read-all":
			return nil, nil
		case "write-all":
			return []Diagnostic{{
				Severity: severity, Path: path,
				Message: "Blanket write permissions grant every available write scope; replace write-all with an explicit least-privilege permission mapping",
			}}, nil
		default:
			return nil, fmt.Errorf("permission-scope requires read-all, write-all, or a permission mapping at %s", path)
		}
	}
	fields, err := policyMapping(value, path)
	if err != nil {
		return nil, err
	}
	var diagnostics []Diagnostic
	var failures []error
	for _, name := range slices.Sorted(maps.Keys(fields)) {
		if err := ctx.Err(); err != nil {
			return diagnostics, errors.Join(append(failures, err)...)
		}
		level, ok := fields[name].(string)
		if !permissionNamePattern.MatchString(name) || !ok || level != "read" && level != "write" && level != "none" {
			failures = append(failures, fmt.Errorf("permission-scope requires literal permission names and read, write, or none levels at %s", path))
			continue
		}
		if name == "id-token" && level == "read" {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: Error, Path: path + ".id-token",
				Message: "OIDC permissions do not support read; use id-token: write only when token issuance is required, otherwise remove the grant",
			})
		}
	}
	return diagnostics, errors.Join(failures...)
}
