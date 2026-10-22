package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

// getTimeoutMinutesCodemod creates a codemod for migrating timeout_minutes to timeout-minutes.
func getTimeoutMinutesCodemod() Codemod {
	return Codemod{
		ID:           "timeout-minutes-migration",
		Name:         "Migrate timeout_minutes to timeout-minutes",
		Description:  "Replaces deprecated 'timeout_minutes' field with 'timeout-minutes'",
		IntroducedIn: "0.1.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Rename([]string{"timeout_minutes"}, "timeout-minutes")
			})
		},
	}
}
