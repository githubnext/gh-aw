package workflow

import "fmt"

// generateGitConfigurationSteps generates standardized git credential setup as string steps
func (c *Compiler) generateGitConfigurationSteps() []string {
	return c.generateGitConfigurationStepsWithToken("${{ github.token }}")
}

// generateGitConfigurationStepsWithToken generates git credential setup with a custom token.
//
// Security Note: This function uses GitHub Actions context variables that are system-provided
// and trusted. Template injection is not a risk here because:
//   - github.repository: Set by GitHub Actions runtime, not user-controllable
//   - github.server_url: Set by GitHub Actions runtime, not user-controllable
//   - token parameter: Either github.token or app-token output, both GitHub-generated
//
// These variables cannot be influenced by user input (PR titles, issue comments, etc.)
// and are safe to use in template expansion contexts.
//
// See docs/src/content/docs/guides/security.md for more information about known false
// positives in security scanning tools.
func (c *Compiler) generateGitConfigurationStepsWithToken(token string) []string {
	return []string{
		"      - name: Configure Git credentials\n",
		"        env:\n",
		"          REPO_NAME: ${{ github.repository }}\n",
		"          SERVER_URL: ${{ github.server_url }}\n",
		"        run: |\n",
		"          git config --global user.email \"github-actions[bot]@users.noreply.github.com\"\n",
		"          git config --global user.name \"github-actions[bot]\"\n",
		"          # Re-authenticate git with GitHub token\n",
		"          SERVER_URL_STRIPPED=\"${SERVER_URL#https://}\"\n",
		fmt.Sprintf("          git remote set-url origin \"https://x-access-token:%s@${SERVER_URL_STRIPPED}/${REPO_NAME}.git\"\n", token),
		"          echo \"Git configured with standard GitHub Actions identity\"\n",
	}
}
