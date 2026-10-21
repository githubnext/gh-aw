package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

// getSafeInputsToMCPScriptsCodemod renames safe-inputs to mcp-scripts.
func getSafeInputsToMCPScriptsCodemod() Codemod {
	return Codemod{
		ID:           "safe-inputs-to-mcp-scripts",
		Name:         "Rename safe-inputs to mcp-scripts",
		Description:  "Renames the top-level 'safe-inputs' key to 'mcp-scripts' in workflow frontmatter",
		IntroducedIn: "0.3.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Rename([]string{"safe-inputs"}, "mcp-scripts")
			})
		},
	}
}
