package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

func getSafeOutputAddReviewerAllowlistsCodemod() Codemod {
	return Codemod{
		ID:           "safe-output-add-reviewer-allowlists",
		Name:         "Rename deprecated add-reviewer allowlist fields",
		Description:  "Renames reviewers to allowed-reviewers and team-reviewers to allowed-team-reviewers in safe-outputs.add-reviewer.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				for _, key := range []string{"reviewers", "team-reviewers"} {
					if err := f.Rename([]string{"safe-outputs", "add-reviewer", key}, "allowed-"+key); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
}
