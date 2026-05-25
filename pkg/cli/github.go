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
// Repositories under the github, githubnext, and microsoft organizations are
// fetched from public GitHub (https://github.com) in cross-host contexts.
// microsoft/* is included because canonical shared workflows (for example
// microsoft/apm/.github/workflows/shared/apm.md) are maintained on github.com.
// For all other repositories, it uses getGitHubHost().
func getGitHubHostForRepo(repo string) string {
	parts := strings.SplitN(repo, "/", 2)
	switch parts[0] {
	case "github", "githubnext", "microsoft":
		githubLog.Printf("Using public GitHub host for %s organization repository", parts[0])
		return string(constants.PublicGitHubHost)
	}

	// For all other repositories, use the configured GitHub host
	return getGitHubHost()
}
