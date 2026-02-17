package cli

import (
	"github.com/github/gh-aw/pkg/gitutil"
)

// getGitHubHost returns the GitHub host URL from environment variables.
// Uses the centralized gitutil.GetGitHubHost() function.
func getGitHubHost() string {
	return gitutil.GetGitHubHost()
}
