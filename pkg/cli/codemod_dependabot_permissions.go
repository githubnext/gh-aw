package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
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

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return ensureToolsetPermissions(lines, missingPermissions)
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

func ensureToolsetPermissions(lines []string, missing map[workflow.PermissionScope]workflow.PermissionLevel) ([]string, bool) {
	if len(missing) == 0 {
		return lines, false
	}

	permissionsIdx := -1
	permissionsIndent := ""
	permissionsEnd := len(lines)

	for i, line := range lines {
		if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "permissions:") {
			permissionsIdx = i
			permissionsIndent = getIndentation(line)
			for j := i + 1; j < len(lines); j++ {
				if isTopLevelKey(lines[j]) {
					permissionsEnd = j
					break
				}
			}
			break
		}
	}

	if permissionsIdx == -1 {
		insertAt := findPermissionsInsertIndex(lines)
		block := []string{"permissions:"}
		for _, key := range sortedMissingPermissionKeys(missing) {
			block = append(block, fmt.Sprintf("  %s: %s", key, missing[workflow.PermissionScope(key)]))
		}

		result := make([]string, 0, len(lines)+len(block))
		result = append(result, lines[:insertAt]...)
		result = append(result, block...)
		result = append(result, lines[insertAt:]...)
		return result, true
	}

	trimmedPermissionsLine := strings.TrimSpace(lines[permissionsIdx])
	inlineValue := strings.TrimSpace(strings.TrimPrefix(trimmedPermissionsLine, "permissions:"))
	if inlineValue != "" && !strings.HasPrefix(inlineValue, "#") {
		block := []string{"permissions:"}
		for _, key := range sortedMissingPermissionKeys(missing) {
			block = append(block, fmt.Sprintf("  %s: %s", key, missing[workflow.PermissionScope(key)]))
		}
		result := make([]string, 0, len(lines)+len(block))
		result = append(result, lines[:permissionsIdx]...)
		result = append(result, block...)
		result = append(result, lines[permissionsIdx+1:]...)
		return result, true
	}

	updated := make([]string, len(lines))
	copy(updated, lines)
	modified := false
	remaining := make(map[string]workflow.PermissionLevel, len(missing))
	for scope, level := range missing {
		remaining[string(scope)] = level
	}

	for i := permissionsIdx + 1; i < permissionsEnd; i++ {
		trimmed := strings.TrimSpace(updated[i])
		key := parseYAMLMapKey(trimmed)
		if key == "" {
			continue
		}
		requiredLevel, needsUpdate := remaining[key]
		if !needsUpdate {
			continue
		}
		level := strings.TrimSpace(strings.TrimPrefix(trimmed, key+":"))
		if level == string(requiredLevel) || level == string(workflow.PermissionWrite) {
			delete(remaining, key)
			continue
		}
		updated[i] = fmt.Sprintf("%s  %s: %s", permissionsIndent, key, requiredLevel)
		delete(remaining, key)
		modified = true
	}

	if len(remaining) == 0 {
		return updated, modified
	}

	insertLines := make([]string, 0, len(remaining))
	for _, key := range sortedRemainingPermissionKeys(remaining) {
		insertLines = append(insertLines, fmt.Sprintf("%s  %s: %s", permissionsIndent, key, remaining[key]))
	}

	result := make([]string, 0, len(updated)+len(insertLines))
	result = append(result, updated[:permissionsEnd]...)
	result = append(result, insertLines...)
	result = append(result, updated[permissionsEnd:]...)
	return result, true
}

func findPermissionsInsertIndex(lines []string) int {
	onIdx := -1
	onEnd := len(lines)
	for i, line := range lines {
		if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "on:") {
			onIdx = i
			for j := i + 1; j < len(lines); j++ {
				if isTopLevelKey(lines[j]) {
					onEnd = j
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
	for scope := range missing {
		keys = append(keys, string(scope))
	}
	sort.Strings(keys)
	return keys
}

func sortedRemainingPermissionKeys(remaining map[string]workflow.PermissionLevel) []string {
	keys := make([]string, 0, len(remaining))
	for key := range remaining {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
