package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var dependabotPermissionsCodemodLog = logger.New("cli:codemod_dependabot_permissions")

// getDependabotPermissionsCodemod ensures vulnerability-alerts: read is present for dependabot toolset usage.
func getDependabotPermissionsCodemod() Codemod {
	return Codemod{
		ID:           "dependabot-toolset-permissions",
		Name:         "Add missing dependabot permissions",
		Description:  "Adds permissions.vulnerability-alerts: read when tools.github.toolsets includes dependabot",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasDependabotToolset(frontmatter) || !needsDependabotPermissionFix(frontmatter) {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, ensureDependabotPermission)
			if applied {
				dependabotPermissionsCodemodLog.Print("Added permissions.vulnerability-alerts: read for dependabot toolset")
			}
			return newContent, applied, err
		},
	}
}

func hasDependabotToolset(frontmatter map[string]any) bool {
	toolsAny, hasTools := frontmatter["tools"]
	if !hasTools {
		return false
	}
	toolsMap, ok := toolsAny.(map[string]any)
	if !ok {
		return false
	}
	githubAny, hasGitHub := toolsMap["github"]
	if !hasGitHub {
		return false
	}
	githubMap, ok := githubAny.(map[string]any)
	if !ok {
		return false
	}
	toolsetsAny, hasToolsets := githubMap["toolsets"]
	if !hasToolsets {
		return false
	}

	switch toolsets := toolsetsAny.(type) {
	case []string:
		for _, toolset := range toolsets {
			if strings.TrimSpace(toolset) == "dependabot" {
				return true
			}
		}
	case []any:
		for _, entry := range toolsets {
			toolset, ok := entry.(string)
			if ok && strings.TrimSpace(toolset) == "dependabot" {
				return true
			}
		}
	case string:
		for toolset := range strings.SplitSeq(toolsets, ",") {
			if strings.TrimSpace(toolset) == "dependabot" {
				return true
			}
		}
	}

	return false
}

func needsDependabotPermissionFix(frontmatter map[string]any) bool {
	permissionsAny, hasPermissions := frontmatter["permissions"]
	if !hasPermissions {
		return true
	}

	switch permissions := permissionsAny.(type) {
	case string:
		trimmed := strings.TrimSpace(permissions)
		return trimmed != "read-all" && trimmed != "write-all"
	case map[string]any:
		levelAny, hasVulnerabilityAlerts := permissions["vulnerability-alerts"]
		if !hasVulnerabilityAlerts {
			return true
		}
		level, ok := levelAny.(string)
		if !ok {
			return true
		}
		trimmed := strings.TrimSpace(level)
		return trimmed != "read" && trimmed != "write"
	default:
		return true
	}
}

func ensureDependabotPermission(lines []string) ([]string, bool) {
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
		block := []string{
			"permissions:",
			"  vulnerability-alerts: read",
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
		block := []string{
			"permissions:",
			"  vulnerability-alerts: read",
		}
		result := make([]string, 0, len(lines)+1)
		result = append(result, lines[:permissionsIdx]...)
		result = append(result, block...)
		result = append(result, lines[permissionsIdx+1:]...)
		return result, true
	}

	permissionKeyPrefix := strings.TrimSpace(permissionsIndent + "  vulnerability-alerts:")
	for i := permissionsIdx + 1; i < permissionsEnd; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if afterPrefix, ok := strings.CutPrefix(trimmed, permissionKeyPrefix); ok {
			level := strings.TrimSpace(afterPrefix)
			if level == "read" || level == "write" {
				return lines, false
			}

			updated := make([]string, len(lines))
			copy(updated, lines)
			updated[i] = permissionsIndent + "  vulnerability-alerts: read"
			return updated, true
		}
	}

	insertedLine := permissionsIndent + "  vulnerability-alerts: read"
	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:permissionsEnd]...)
	result = append(result, insertedLine)
	result = append(result, lines[permissionsEnd:]...)
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
