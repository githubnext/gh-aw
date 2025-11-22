package workflow

import "fmt"

// generateGitConfigurationSteps generates standardized git credential setup as string steps
func (c *Compiler) generateGitConfigurationSteps() []string {
	return c.generateGitConfigurationStepsWithToken("${{ github.token }}")
}

// generateGitConfigurationStepsWithToken generates git credential setup with a custom token
func (c *Compiler) generateGitConfigurationStepsWithToken(token string) []string {
	return []string{
		"      - name: Configure Git credentials\n",
		"        env:\n",
		"          REPO_NAME: ${{ github.repository }}\n",
		"        run: |\n",
		"          git config --global user.email \"github-actions[bot]@users.noreply.github.com\"\n",
		"          git config --global user.name \"github-actions[bot]\"\n",
		"          # Re-authenticate git with GitHub token\n",
		"          SERVER_URL=\"${{ github.server_url }}\"\n",
		"          SERVER_URL=\"${SERVER_URL#https://}\"\n",
		fmt.Sprintf("          git remote set-url origin \"https://x-access-token:%s@${SERVER_URL}/${REPO_NAME}.git\"\n", token),
		"          echo \"Git configured with standard GitHub Actions identity\"\n",
	}
}
