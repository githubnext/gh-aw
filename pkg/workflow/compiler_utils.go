package workflow

import (
	"strings"
)

// generateSafeFileName converts a workflow name to a safe filename for logs
func generateSafeFileName(name string) string {
	// Replace spaces and special characters with hyphens
	result := strings.ReplaceAll(name, " ", "-")
	result = strings.ReplaceAll(result, "/", "-")
	result = strings.ReplaceAll(result, "\\", "-")
	result = strings.ReplaceAll(result, ":", "-")
	result = strings.ReplaceAll(result, "*", "-")
	result = strings.ReplaceAll(result, "?", "-")
	result = strings.ReplaceAll(result, "\"", "-")
	result = strings.ReplaceAll(result, "<", "-")
	result = strings.ReplaceAll(result, ">", "-")
	result = strings.ReplaceAll(result, "|", "-")
	result = strings.ReplaceAll(result, "@", "-")
	result = strings.ToLower(result)

	// Remove multiple consecutive hyphens
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim leading/trailing hyphens
	result = strings.Trim(result, "-")

	// Ensure it's not empty
	if result == "" {
		result = "workflow"
	}

	return result
}

// extractToolsFromFrontmatter extracts the tools configuration from frontmatter
func extractToolsFromFrontmatter(frontmatter map[string]any) map[string]any {
	tools := make(map[string]any)
	if toolsValue, exists := frontmatter["tools"]; exists {
		if toolsMap, ok := toolsValue.(map[string]any); ok {
			tools = toolsMap
		}
	}
	return tools
}

// getGitHubDockerImageVersion extracts the docker image version from GitHub tool config
func getGitHubDockerImageVersion(githubTool any) string {
	githubDockerImageVersion := "sha-45e90ae" // Default Docker image version
	// Extract docker_image_version setting from tool properties
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if versionSetting, exists := toolConfig["docker_image_version"]; exists {
			if stringValue, ok := versionSetting.(string); ok {
				githubDockerImageVersion = stringValue
			}
		}
	}
	return githubDockerImageVersion
}
