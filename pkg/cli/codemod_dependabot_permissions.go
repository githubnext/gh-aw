package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var dependabotPermissionsCodemodLog = logger.New("cli:codemod_dependabot_permissions")

// getDependabotPermissionsCodemod ensures required read permissions are present for declared GitHub toolsets.
func getDependabotPermissionsCodemod() Codemod {
	return Codemod{
		ID:           "dependabot-toolset-permissions",
		Name:         "Add missing GitHub toolset permissions",
		Description:  "Adds missing permissions.<scope>: read when tools.github.toolsets requires them",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			toolsets, ok := extractGitHubToolsets(frontmatter)
			if !ok || len(toolsets) == 0 {
				return content, false, nil
			}

			missingPermissions := findMissingToolsetPermissions(frontmatter, toolsets)
			if len(missingPermissions) == 0 {
				return content, false, nil
			}

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				if value, exists := f.Lookup("permissions"); exists {
					if _, ok := value.(map[string]any); !ok {
						if err := f.Set([]string{"permissions"}, map[string]any{}); err != nil {
							return err
						}
					}
				}
				for _, key := range sortedMissingPermissionKeys(missingPermissions) {
					if err := f.Set([]string{"permissions", key}, string(missingPermissions[workflow.PermissionScope(key)])); err != nil {
						return err
					}
				}
				return nil
			})
			if applied {
				dependabotPermissionsCodemodLog.Printf("Added missing permissions for GitHub toolsets: %v", sortedMissingPermissionKeys(missingPermissions))
			}
			return newContent, applied, err
		},
	}
}

func extractGitHubToolsets(frontmatter map[string]any) ([]string, bool) {
	toolsAny, hasTools := frontmatter["tools"]
	if !hasTools {
		return nil, false
	}
	toolsMap, ok := toolsAny.(map[string]any)
	if !ok {
		return nil, false
	}
	githubAny, hasGitHub := toolsMap["github"]
	if !hasGitHub {
		return nil, false
	}
	githubMap, ok := githubAny.(map[string]any)
	if !ok {
		return nil, false
	}
	toolsetsAny, hasToolsets := githubMap["toolsets"]
	if !hasToolsets {
		return nil, false
	}

	var toolsets []string
	switch configured := toolsetsAny.(type) {
	case []string:
		for _, toolset := range configured {
			trimmed := strings.TrimSpace(toolset)
			if trimmed != "" {
				toolsets = append(toolsets, trimmed)
			}
		}
	case []any:
		for _, entry := range configured {
			toolset, ok := entry.(string)
			if ok {
				trimmed := strings.TrimSpace(toolset)
				if trimmed != "" {
					toolsets = append(toolsets, trimmed)
				}
			}
		}
	case string:
		for toolset := range strings.SplitSeq(configured, ",") {
			trimmed := strings.TrimSpace(toolset)
			if trimmed != "" {
				toolsets = append(toolsets, trimmed)
			}
		}
	}

	return toolsets, len(toolsets) > 0
}

func findMissingToolsetPermissions(frontmatter map[string]any, toolsets []string) map[workflow.PermissionScope]workflow.PermissionLevel {
	permissionsParser := workflow.NewPermissionsParserFromValue(frontmatter["permissions"])
	currentPermissions := permissionsParser.ToPermissions()
	validationResult := workflow.ValidatePermissions(currentPermissions, &workflow.GitHubToolConfig{}, toolsets)
	if !validationResult.HasValidationIssues || len(validationResult.MissingPermissions) == 0 {
		return nil
	}

	missing := make(map[workflow.PermissionScope]workflow.PermissionLevel)
	for scope, level := range validationResult.MissingPermissions {
		// Skip GitHub App-only scopes: these are not grantable through workflow
		// GITHUB_TOKEN permissions and require GitHub App token minting instead.
		if workflow.IsGitHubAppOnlyScope(scope) {
			continue
		}
		missing[scope] = level
	}
	return missing
}

func findPermissionsInsertIndex(lines []string) int {
	onIdx := -1
	onEnd := len(lines)
	for i, line := range lines {
		if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "on:") {
			onIdx = i
			for j, next := range lines[i+1:] {
				if isTopLevelKey(next) {
					onEnd = i + 1 + j
					break
				}
			}
			break
		}
	}

	if onIdx >= 0 {
		return onEnd
	}

	return 0
}

func sortedMissingPermissionKeys(missing map[workflow.PermissionScope]workflow.PermissionLevel) []string {
	keys := make([]string, 0, len(missing))
	for _, scope := range sliceutil.SortedKeys(missing) {
		keys = append(keys, string(scope))
	}
	return keys
}
