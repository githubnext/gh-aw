package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

func getSafeOutputMergePRConstraintsCodemod() Codemod {
	return Codemod{
		ID:           "safe-output-merge-pr-constraints",
		Name:         "Rename deprecated merge-pull-request constraint fields",
		Description:  "Renames allowed-labels to required-labels in safe-outputs.merge-pull-request.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Rename([]string{"safe-outputs", "merge-pull-request", "allowed-labels"}, "required-labels")
			})
		},
	}
}
