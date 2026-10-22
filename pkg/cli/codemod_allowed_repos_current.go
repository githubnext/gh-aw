package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

func getAllowedReposCurrentToGitHubRepositoryCodemod() Codemod {
	return Codemod{
		ID:           "allowed-repos-current-to-github-repository",
		Name:         "Migrate 'tools.github.allowed-repos: current' to '${{ github.repository }}'",
		Description:  "Rewrites the legacy 'current' alias for tools.github.allowed-repos to the accepted '${{ github.repository }}' expression.",
		IntroducedIn: "0.85.5",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				path := []string{"tools", "github", "allowed-repos"}
				if value, _ := f.Lookup(path...); value == "current" {
					return f.Set(path, "${{ github.repository }}")
				}
				return nil
			})
		},
	}
}
