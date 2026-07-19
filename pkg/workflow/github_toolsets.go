package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
)

var toolsetsLog = logger.New("workflow:github_toolsets")

// DefaultGitHubToolsets defines the toolsets that are enabled by default
// when toolsets are not explicitly specified in the GitHub MCP configuration.
// These match the documented default toolsets in github-mcp-server.md
var DefaultGitHubToolsets = []string{"context", "repos", "issues", "pull_requests"}

// ActionFriendlyGitHubToolsets defines the default toolsets that work with GitHub Actions tokens.
// This excludes "users" toolset because GitHub Actions tokens do not support user operations.
// Use this when the workflow will run in GitHub Actions with GITHUB_TOKEN.
var ActionFriendlyGitHubToolsets = []string{"context", "repos", "issues", "pull_requests"}

// GitHubToolsetsExcludedFromAll defines toolsets that are NOT included when "all" is specified.
// These toolsets are opt-in only to avoid granting unnecessary permissions by default.
var GitHubToolsetsExcludedFromAll = []string{"dependabot"}

// ParseGitHubToolsets parses the toolsets string and expands "default" and "all"
// into their constituent toolsets. It handles comma-separated lists and deduplicates.
func ParseGitHubToolsets(toolsetsStr string) []string {
	toolsetsLog.Printf("Parsing GitHub toolsets: %q", toolsetsStr)
	if toolsetsStr == "" {
		toolsetsLog.Printf("Empty toolsets string, using defaults: %v", DefaultGitHubToolsets)
		return DefaultGitHubToolsets
	}
	expander := githubToolsetExpander{seen: make(map[string]struct{})}
	for _, toolset := range strings.Split(toolsetsStr, ",") {
		expander.expand(strings.TrimSpace(toolset))
	}
	toolsetsLog.Printf("Parsed toolsets result: %d unique toolsets expanded from input", len(expander.expanded))
	return expander.expanded
}

type githubToolsetExpander struct {
	expanded []string
	seen     map[string]struct{}
}

func (e *githubToolsetExpander) expand(toolset string) {
	if toolset == "" {
		return
	}
	switch toolset {
	case "default":
		toolsetsLog.Printf("Expanding 'default' to %d toolsets", len(DefaultGitHubToolsets))
		e.addMany(DefaultGitHubToolsets)
	case "action-friendly":
		toolsetsLog.Printf("Expanding 'action-friendly' to %d toolsets", len(ActionFriendlyGitHubToolsets))
		e.addMany(ActionFriendlyGitHubToolsets)
	case "all":
		e.expandAll()
	default:
		e.add(toolset)
	}
}

func (e *githubToolsetExpander) expandAll() {
	toolsetsLog.Printf("Expanding 'all' to toolsets from permissions map (excluding %v)", GitHubToolsetsExcludedFromAll)
	excludedMap := make(map[string]struct{}, len(GitHubToolsetsExcludedFromAll))
	for _, ex := range GitHubToolsetsExcludedFromAll {
		excludedMap[ex] = struct{}{}
	}
	for t := range getToolsetPermissionsMap() {
		if !setutil.Contains(excludedMap, t) {
			e.add(t)
		}
	}
}

func (e *githubToolsetExpander) addMany(toolsets []string) {
	for _, toolset := range toolsets {
		e.add(toolset)
	}
}

func (e *githubToolsetExpander) add(toolset string) {
	if setutil.Contains(e.seen, toolset) {
		return
	}
	e.expanded = append(e.expanded, toolset)
	e.seen[toolset] = struct{}{}
}
