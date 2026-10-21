package cli

import (
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

// getGitHubAppCodemod renames app at the top level and in supported configuration sections.
func getGitHubAppCodemod() Codemod {
	return Codemod{
		ID:           "app-to-github-app",
		Name:         "Rename 'app' to 'github-app'",
		Description:  "Renames the deprecated 'app:' field to 'github-app:' at the top level and in tools.github, safe-outputs, and checkout configurations.",
		IntroducedIn: "0.15.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				for _, path := range [][]string{
					{"app"},
					{"tools", "github", "app"},
					{"safe-outputs", "app"},
					{"checkout", "app"},
				} {
					if err := f.Rename(path, "github-app"); err != nil {
						return err
					}
				}
				// Mapping paths cannot address sequence items; edit their authored AST instead.
				return f.Edit(func(document *aw.Document) error {
					checkouts, ok := document.Frontmatter["checkout"].([]any)
					if !ok {
						return nil
					}
					for _, value := range checkouts {
						checkout, ok := value.(map[string]any)
						if !ok {
							continue
						}
						app, exists := checkout["app"]
						if _, conflict := checkout["github-app"]; exists && !conflict {
							checkout["github-app"] = app
							delete(checkout, "app")
						}
					}
					return nil
				})
			})
		},
	}
}
