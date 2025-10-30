package workflow

import (
	"strings"
)

// DefaultGitHubToolsets defines the toolsets that are enabled by default
// when toolsets are not explicitly specified in the GitHub MCP configuration.
// These match the documented default toolsets in github-mcp-server.instructions.md
var DefaultGitHubToolsets = []string{"context", "repos", "issues", "pull_requests", "users"}

// ParseGitHubToolsets parses the toolsets string and expands "default" and "all"
// into their constituent toolsets. It handles comma-separated lists and deduplicates.
func ParseGitHubToolsets(toolsetsStr string) []string {
	if toolsetsStr == "" {
		return DefaultGitHubToolsets
	}

	toolsets := strings.Split(toolsetsStr, ",")
	var expanded []string
	seenToolsets := make(map[string]bool)

	for _, toolset := range toolsets {
		toolset = strings.TrimSpace(toolset)
		if toolset == "" {
			continue
		}

		if toolset == "default" {
			// Add default toolsets
			for _, dt := range DefaultGitHubToolsets {
				if !seenToolsets[dt] {
					expanded = append(expanded, dt)
					seenToolsets[dt] = true
				}
			}
		} else if toolset == "all" {
			// Add all toolsets from the toolset permissions map
			for t := range toolsetPermissionsMap {
				if !seenToolsets[t] {
					expanded = append(expanded, t)
					seenToolsets[t] = true
				}
			}
		} else {
			// Add individual toolset
			if !seenToolsets[toolset] {
				expanded = append(expanded, toolset)
				seenToolsets[toolset] = true
			}
		}
	}

	return expanded
}
