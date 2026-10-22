package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

// getGitHubReposToAllowedReposCodemod renames tools.github.repos to allowed-repos.
func getGitHubReposToAllowedReposCodemod() Codemod {
	return Codemod{
		ID:           "github-repos-to-allowed-repos",
		Name:         "Rename 'tools.github.repos' to 'tools.github.allowed-repos'",
		Description:  "Renames the deprecated 'repos:' field to 'allowed-repos:' inside the tools.github configuration block.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Rename([]string{"tools", "github", "repos"}, "allowed-repos")
			})
		},
	}
}
