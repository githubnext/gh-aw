package workflow

// getEffectiveGitHubToken returns the GitHub token to use, with precedence:
// 1. Custom token passed as parameter (e.g., from safe-outputs)
// 2. Top-level github-token from frontmatter
// 3. Default fallback: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
func getEffectiveGitHubToken(customToken, toplevelToken string) string {
	if customToken != "" {
		return customToken
	}
	if toplevelToken != "" {
		return toplevelToken
	}
	return "${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}"
}

// getEffectiveCopilotGitHubToken returns the GitHub token to use for Copilot-related operations,
// with precedence:
// 1. Custom token passed as parameter (e.g., from safe-outputs config github-token field)
// 2. secrets.GH_AW_COPILOT_TOKEN (special token for Copilot operations like assigning copilot, creating agent tasks)
// 3. secrets.GH_AW_GITHUB_TOKEN (general GitHub token)
// 4. secrets.GITHUB_TOKEN (default GitHub Actions token)
// This is used for safe outputs that interact with GitHub Copilot features:
// - create-agent-task
// - assigning "copilot" to issues
// - adding "copilot" as PR reviewer
func getEffectiveCopilotGitHubToken(customToken, toplevelToken string) string {
	if customToken != "" {
		return customToken
	}
	if toplevelToken != "" {
		return toplevelToken
	}
	return "${{ secrets.GH_AW_COPILOT_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}"
}
