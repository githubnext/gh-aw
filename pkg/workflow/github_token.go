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
