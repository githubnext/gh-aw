package workflow

import (
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var compilerYamlHelpersLog = logger.New("workflow:compiler_yaml_helpers")

// ContainsCheckout returns true if the given custom steps contain an actions/checkout step
func ContainsCheckout(customSteps string) bool {
	if customSteps == "" {
		return false
	}

	// Look for actions/checkout usage patterns
	checkoutPatterns := []string{
		"actions/checkout@",
		"uses: actions/checkout",
		"- uses: actions/checkout",
	}

	lowerSteps := strings.ToLower(customSteps)
	for _, pattern := range checkoutPatterns {
		if strings.Contains(lowerSteps, strings.ToLower(pattern)) {
			compilerYamlHelpersLog.Print("Detected actions/checkout in custom steps")
			return true
		}
	}

	return false
}

// GetWorkflowIDFromPath extracts the workflow ID from a markdown file path.
// The workflow ID is the filename without the .md extension.
// Example: "/path/to/ai-moderator.md" -> "ai-moderator"
func GetWorkflowIDFromPath(markdownPath string) string {
	return strings.TrimSuffix(filepath.Base(markdownPath), ".md")
}

// generateGitHubScriptWithRequire is implemented in compiler_github_actions_steps.go

// generateInlineGitHubScriptStep is implemented in compiler_github_actions_steps.go
