package workflow

import "strings"

// generateGitHubContextPromptStep generates a separate step for GitHub context information
// when the github tool is enabled. This injects repository, issue, discussion, pull request,
// comment, and run ID information into the prompt.
//
// The function uses generateStaticPromptStepWithExpressions to securely handle the GitHub
// Actions expressions in the context prompt. This extracts ${{ ... }} expressions into
// environment variables and uses shell variable expansion in the heredoc, preventing
// template injection vulnerabilities.
func (c *Compiler) generateGitHubContextPromptStep(yaml *strings.Builder, data *WorkflowData) {
	generateStaticPromptStepWithExpressions(yaml,
		"Append GitHub context to prompt",
		githubContextPromptText,
		hasGitHubTool(data.ParsedTools))
}
