package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

func getSafeOutputRequireTitlePrefixCodemod() Codemod {
	return Codemod{
		ID:           "safe-output-title-prefix-to-required-title-prefix",
		Name:         "Rename deprecated safe-outputs title-prefix constraints",
		Description:  "Renames deprecated constraint fields to required-title-prefix/required-labels for applicable safe-outputs handlers.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				for _, handler := range []string{"close-issue", "close-pull-request", "close-discussion", "mark-pull-request-as-ready-for-review", "push-to-pull-request-branch"} {
					if err := f.Rename([]string{"safe-outputs", handler, "title-prefix"}, "required-title-prefix"); err != nil {
						return err
					}
				}
				return f.Rename([]string{"safe-outputs", "push-to-pull-request-branch", "labels"}, "required-labels")
			})
		},
	}
}
