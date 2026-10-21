package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

// getMaxRunsToMaxTurnsCodemod migrates deprecated top-level max-runs to max-turns.
func getMaxRunsToMaxTurnsCodemod() Codemod {
	return Codemod{
		ID:           "max-runs-to-max-turns",
		Name:         "Rename top-level max-runs to max-turns",
		Description:  "Renames deprecated top-level 'max-runs' to top-level 'max-turns'. If both exist, preserves 'max-turns' and removes 'max-runs'.",
		IntroducedIn: "1.0.76",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				if _, exists := f.Lookup("max-turns"); exists {
					return f.Remove("max-runs")
				}
				return f.Rename([]string{"max-runs"}, "max-turns")
			})
		},
	}
}
