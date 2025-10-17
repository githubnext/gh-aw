package workflow

import "strings"

// generateGitHubContextPromptStep generates a separate step for GitHub context information
// when the github tool is enabled. This injects repository, issue, discussion, pull request,
// comment, and run ID information into the prompt.
func (c *Compiler) generateGitHubContextPromptStep(yaml *strings.Builder, data *WorkflowData) {
	generateStaticPromptStep(yaml,
		"Append GitHub context to prompt",
		githubContextPromptText,
		hasGitHubTool(data.Tools))
}
