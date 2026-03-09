package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var gitConfigStepsLog = logger.New("workflow:git_configuration_steps")

// generateGitConfigurationSteps generates standardized git credential setup as string steps
func (c *Compiler) generateGitConfigurationSteps() []string {
	return c.generateGitConfigurationStepsWithToken("${{ github.token }}", "")
}

// generateGitConfigurationStepsWithToken generates git credential setup with a custom token
// and optional target repository for cross-repo operations
// Parameters:
//   - token: GitHub token to use for authentication
//   - targetRepoSlug: optional target repository (e.g., "org/repo") for cross-repo operations
//     If empty, uses source repository (github.repository)
//     If set, configures git remote to point to the target repository
func (c *Compiler) generateGitConfigurationStepsWithToken(token string, targetRepoSlug string) []string {
	// Determine which repository to configure git remote for
	// Priority: targetRepoSlug > trialLogicalRepoSlug > default (source repo)
	repoNameValue := "${{ github.repository }}"
	if targetRepoSlug != "" {
		repoNameValue = fmt.Sprintf("%q", targetRepoSlug)
		gitConfigStepsLog.Printf("Generating git config steps with target repo: %s", targetRepoSlug)
	} else if c.trialMode && c.trialLogicalRepoSlug != "" {
		repoNameValue = fmt.Sprintf("%q", c.trialLogicalRepoSlug)
		gitConfigStepsLog.Printf("Generating git config steps in trial mode with logical repo: %s", c.trialLogicalRepoSlug)
	} else {
		gitConfigStepsLog.Print("Generating git config steps with default github.repository")
	}

	return []string{
		"      - name: Configure Git credentials\n",
		"        env:\n",
		fmt.Sprintf("          REPO_NAME: %s\n", repoNameValue),
		"          SERVER_URL: ${{ github.server_url }}\n",
		"        run: |\n",
		"          git config --global user.email \"github-actions[bot]@users.noreply.github.com\"\n",
		"          git config --global user.name \"github-actions[bot]\"\n",
		"          git config --global am.keepcr true\n",
		"          # Re-authenticate git with GitHub token\n",
		"          SERVER_URL_STRIPPED=\"${SERVER_URL#https://}\"\n",
		fmt.Sprintf("          git remote set-url origin \"https://x-access-token:%s@${SERVER_URL_STRIPPED}/${REPO_NAME}.git\"\n", token),
		"          echo \"Git configured with standard GitHub Actions identity\"\n",
	}
}

// getGitIdentityEnvVars returns a map of git identity environment variables.
// These mirror the values set by generateGitConfigurationSteps so that git commits
// work correctly inside the AWF sandbox container, which cannot read the host-side
// ~/.gitconfig written by "Configure Git credentials".
//
// Git environment variables take precedence over gitconfig settings and are forwarded
// into the container by AWF via --env-all, ensuring the first git commit succeeds
// without the agent needing to self-configure.
func getGitIdentityEnvVars() map[string]string {
	return map[string]string{
		"GIT_AUTHOR_NAME":     "github-actions[bot]",
		"GIT_AUTHOR_EMAIL":    "github-actions[bot]@users.noreply.github.com",
		"GIT_COMMITTER_NAME":  "github-actions[bot]",
		"GIT_COMMITTER_EMAIL": "github-actions[bot]@users.noreply.github.com",
	}
}

// generateGitCredentialsCleanerStep generates a step that removes git credentials from .git/config
// This is a security measure to prevent credentials left by injected steps from being accessed by the agent
func (c *Compiler) generateGitCredentialsCleanerStep() []string {
	return []string{
		"      - name: Clean git credentials\n",
		"        run: bash /opt/gh-aw/actions/clean_git_credentials.sh\n",
	}
}
