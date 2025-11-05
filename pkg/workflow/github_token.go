package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var tokenLog = logger.New("workflow:github_token")

// getEffectiveGitHubToken returns the GitHub token to use, with precedence:
// 1. Custom token passed as parameter (e.g., from safe-outputs)
// 2. Top-level github-token from frontmatter
// 3. Default fallback: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
func getEffectiveGitHubToken(customToken, toplevelToken string) string {
	if customToken != "" {
		tokenLog.Print("Using custom GitHub token")
		return customToken
	}
	if toplevelToken != "" {
		tokenLog.Print("Using top-level GitHub token from frontmatter")
		return toplevelToken
	}
	tokenLog.Print("Using default GitHub token fallback")
	return "${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}"
}

// getEffectiveCopilotGitHubToken returns the GitHub token to use for Copilot-related operations,
// with precedence:
// 1. Custom token passed as parameter (e.g., from safe-outputs config github-token field)
// 2. secrets.COPILOT_GITHUB_TOKEN (recommended token for Copilot operations)
// 3. secrets.COPILOT_CLI_TOKEN (alternative recommended token)
// 4. secrets.GH_AW_COPILOT_TOKEN (legacy token for backward compatibility)
// 5. secrets.GH_AW_GITHUB_TOKEN (legacy fallback for backward compatibility)
// Note: The default GITHUB_TOKEN is NOT included as a fallback because it does not have
// permission to create agent tasks, assign issues to bots, or add bots as reviewers.
// This is used for safe outputs that interact with GitHub Copilot features:
// - create-agent-task
// - assigning "copilot" to issues
// - adding "copilot" as PR reviewer
func getEffectiveCopilotGitHubToken(customToken, toplevelToken string) string {
	if customToken != "" {
		tokenLog.Print("Using custom Copilot GitHub token")
		return customToken
	}
	if toplevelToken != "" {
		tokenLog.Print("Using top-level Copilot GitHub token from frontmatter")
		return toplevelToken
	}
	tokenLog.Print("Using default Copilot GitHub token fallback")
	return "${{ secrets.COPILOT_GITHUB_TOKEN || secrets.COPILOT_CLI_TOKEN || secrets.GH_AW_COPILOT_TOKEN || secrets.GH_AW_GITHUB_TOKEN }}"
}
