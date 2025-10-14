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

	yaml.WriteString("      - name: Append GitHub context to prompt\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")
	yaml.WriteString("        run: |\n")
	WritePromptTextToYAML(yaml, githubContextPromptText, "          ")
}
