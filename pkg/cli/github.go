package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var githubLog = logger.New("cli:github")

// getGitHubHost returns the GitHub host URL from environment variables.
// Delegates to parser.GetGitHubHost() for the shared implementation.
func getGitHubHost() string {
	return parser.GetGitHubHost()
}

// getGitHubHostForRepo returns the GitHub host URL for a specific repository.
// The gh-aw repository (github/gh-aw) and the agentics workflow library
// (githubnext/agentics) always use public GitHub (https://github.com)
// regardless of enterprise GitHub host settings, since these repositories are
// only available on public GitHub. For all other repositories, it uses getGitHubHost().
func getGitHubHostForRepo(repo string) string {
	// The gh-aw repository is always on public GitHub
	if repo == "github/gh-aw" || strings.HasPrefix(repo, "github/gh-aw/") {
		githubLog.Print("Using public GitHub host for github/gh-aw repository")
		return string(constants.PublicGitHubHost)
	}

	// The agentics workflow library is always on public GitHub
	if repo == "githubnext/agentics" || strings.HasPrefix(repo, "githubnext/agentics/") {
		githubLog.Print("Using public GitHub host for githubnext/agentics repository")
		return string(constants.PublicGitHubHost)
	}

	// For all other repositories, use the configured GitHub host
	return getGitHubHost()
}
