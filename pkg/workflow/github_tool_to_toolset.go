package workflow

import (
	_ "embed"
	"encoding/json"

	"github.com/github/gh-aw/pkg/logger"
)

var githubToolToToolsetLog = logger.New("workflow:github_tool_to_toolset")

//go:embed data/github_tool_to_toolset.json
var githubToolToToolsetJSON []byte

// GitHubToolToToolsetMap maps individual GitHub MCP tools to their respective toolsets
// This mapping is loaded from an embedded JSON file based on the documentation
// in .github/aw/github-mcp-server.md
var GitHubToolToToolsetMap map[string]string

func init() {
	// Load the mapping from embedded JSON
	if err := json.Unmarshal(githubToolToToolsetJSON, &GitHubToolToToolsetMap); err != nil {
		panic("failed to load GitHub tool to toolset mapping: " + err.Error())
	}
}

// GitHubToolToToolsetMap is the last declaration in this file; ValidateGitHubToolsAgainstToolsets
// has been moved to tools_validation.go.
