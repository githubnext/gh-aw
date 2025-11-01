package workflow

// generateGitConfigurationSteps generates standardized git credential setup as string steps
func (c *Compiler) generateGitConfigurationSteps() []string {
	return []string{
		"      - name: Configure Git credentials\n",
		"        env:\n",
		"          REPO_NAME: ${{ github.repository }}\n",
		"          SERVER_URL: ${{ github.server_url }}\n",
		"          GITHUB_TOKEN: ${{ github.token }}\n",
		"        run: |\n",
		"          git config --global user.email \"github-actions[bot]@users.noreply.github.com\"\n",
		"          git config --global user.name \"github-actions[bot]\"\n",
		"          # Re-authenticate git with GitHub token\n",
		"          SERVER_URL_STRIPPED=\"${SERVER_URL#https://}\"\n",
		"          git remote set-url origin \"https://x-access-token:${GITHUB_TOKEN}@${SERVER_URL_STRIPPED}/${REPO_NAME}.git\"\n",
		"          echo \"Git configured with standard GitHub Actions identity\"\n",
	}
}
