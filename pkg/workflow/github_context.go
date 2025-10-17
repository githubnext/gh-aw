package workflow

import "strings"

// generateGitHubContextPromptStep generates a separate step for GitHub context information
// when the github tool is enabled. This injects repository, issue, discussion, pull request,
// comment, and run ID information into the prompt.
func (c *Compiler) generateGitHubContextPromptStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if GitHub tool is enabled
	if !hasGitHubTool(data.Tools) {
		return // No GitHub tool, skip context injection
	}

	appendPromptStep(yaml,
		"Append GitHub context to prompt",
		func(y *strings.Builder, indent string) {
			WritePromptTextToYAML(y, githubContextPromptText, indent)
		},
		"", // no condition
		"          ")
}
