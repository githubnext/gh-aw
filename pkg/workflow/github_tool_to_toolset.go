package workflow

import (
	_ "embed"
	"encoding/json"
)

//go:embed data/github_tool_to_toolset.json
var githubToolToToolsetJSON []byte

// GitHubToolToToolsetMap maps individual GitHub MCP tools to their respective toolsets
// This mapping is loaded from an embedded JSON file based on the documentation
// in .github/instructions/github-mcp-server.instructions.md
var GitHubToolToToolsetMap map[string]string

func init() {
	// Load the mapping from embedded JSON
	if err := json.Unmarshal(githubToolToToolsetJSON, &GitHubToolToToolsetMap); err != nil {
		panic("failed to load GitHub tool to toolset mapping: " + err.Error())
	}
}

// ValidateGitHubToolsAgainstToolsets validates that all allowed GitHub tools have their
// corresponding toolsets enabled in the configuration
func ValidateGitHubToolsAgainstToolsets(allowedTools []string, enabledToolsets []string) error {
	if len(allowedTools) == 0 {
		// No specific tools restricted, validation not needed
		return nil
	}

	// Create a set of enabled toolsets for fast lookup
	enabledSet := make(map[string]bool)
	for _, toolset := range enabledToolsets {
		enabledSet[toolset] = true
	}

	// Track missing toolsets and which tools need them
	missingToolsets := make(map[string][]string) // toolset -> list of tools that need it

	for _, tool := range allowedTools {
		requiredToolset, exists := GitHubToolToToolsetMap[tool]
		if !exists {
			// Tool not in our mapping - this could be a new tool or a typo
			// We'll skip validation for unknown tools to avoid false positives
			continue
		}

		if !enabledSet[requiredToolset] {
			missingToolsets[requiredToolset] = append(missingToolsets[requiredToolset], tool)
		}
	}

	if len(missingToolsets) > 0 {
		return NewGitHubToolsetValidationError(missingToolsets)
	}

	return nil
}
