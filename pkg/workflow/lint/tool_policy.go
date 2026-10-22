package lint

import (
	"context"
	"errors"
	"fmt"
)

func ToolScopeRule() Rule {
	return Rule{
		ID: "tool-scope", Description: "Check explicit shell/proxy policy conflicts and repository-wide cache scope",
		ThreatRuleID: "CTR-003",
		Severity:     Warning, Targets: AWAST, Check: checkToolScopes,
	}
}

func checkToolScopes(ctx context.Context, input Input) ([]Diagnostic, error) {
	tools, err := authoredPolicyMapping(ctx, input, "tools")
	if err != nil || tools == nil {
		return nil, err
	}
	severity, err := sourcePolicySeverity(input, "tool-scope")
	if err != nil {
		return nil, err
	}
	diagnostics, err := shellToolPolicy(ctx, input, tools, severity)
	failures := []error{err}
	var caches []any
	switch config := tools["cache-memory"].(type) {
	case nil, bool:
	case []any:
		caches = config
	case map[string]any:
		caches = []any{config}
	default:
		failures = append(failures, errors.New("tool-scope requires a cache-memory mapping, list, or boolean"))
	}
	for index, raw := range caches {
		if err := ctx.Err(); err != nil {
			return diagnostics, errors.Join(append(failures, err)...)
		}
		path := "tools.cache-memory"
		if _, list := tools["cache-memory"].([]any); list {
			path = fmt.Sprintf("%s[%d]", path, index)
		}
		cache, err := policyMapping(raw, path)
		failures = append(failures, err)
		if scope, exists := cache["scope"]; exists {
			text, ok := scope.(string)
			if !ok {
				failures = append(failures, fmt.Errorf("tool-scope requires a literal cache scope at %s.scope", path))
			} else if text == "repo" {
				diagnostics = append(diagnostics, Diagnostic{Severity: severity, Path: path + ".scope",
					Message: "Repository-wide cache memory crosses workflow boundaries; use workflow-scoped cache entries"})
			}
		}
	}
	return diagnostics, errors.Join(failures...)
}

func shellToolPolicy(ctx context.Context, input Input, tools map[string]any, severity Severity) ([]Diagnostic, error) {
	var github map[string]any
	var err error
	switch raw := tools["github"].(type) {
	case nil, bool:
	default:
		github, err = policyMapping(raw, "tools.github")
		if err != nil {
			return nil, err
		}
	}
	pending, err := authoredImportsPending(input)
	if err != nil {
		return nil, err
	}
	var diagnostics []Diagnostic
	if github["min-integrity"] == "none" {
		if _, specified := tools["bash"]; !specified && !pending {
			diagnostics = append(diagnostics, Diagnostic{Severity: severity, Path: "tools.bash",
				Message: "GitHub min-integrity none requires an explicit shell-access policy; configure tools.bash"})
		}
	}
	refused := false
	switch bash := tools["bash"].(type) {
	case bool:
		refused = !bash
	case []any, []string:
		commands, err := policyStrings(ctx, bash, "tools.bash")
		if err != nil {
			return diagnostics, err
		}
		refused = len(commands) == 0
	}
	if refused {
		if proxy, exists := tools["cli-proxy"]; (exists || !pending) && proxy != false {
			diagnostics = append(diagnostics, Diagnostic{Severity: severity, Path: "tools.cli-proxy",
				Message: "Refused shell access cannot support CLI-mounted tools; explicitly disable tools.cli-proxy"})
		}
		if github["mode"] == "gh-proxy" {
			diagnostics = append(diagnostics, Diagnostic{Severity: severity, Path: "tools.github.mode",
				Message: "GitHub gh-proxy requires shell access; use an MCP-backed mode when bash is refused"})
		}
	}
	return diagnostics, ctx.Err()
}
