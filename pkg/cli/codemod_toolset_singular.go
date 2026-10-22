package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

// getToolsetSingularToToolsetsCodemod corrects tools.github.toolset to toolsets.
func getToolsetSingularToToolsetsCodemod() Codemod {
	return Codemod{
		ID:           "toolset-singular-to-toolsets",
		Name:         "Rename 'tools.github.toolset' to 'tools.github.toolsets'",
		Description:  "Renames the mistyped singular 'toolset:' field to the correct plural 'toolsets:' inside the tools.github configuration block.",
		IntroducedIn: "0.85.5",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Rename([]string{"tools", "github", "toolset"}, "toolsets")
			})
		},
	}
}
