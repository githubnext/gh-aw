package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

func getMentionsAllowTeamMembersCodemod() Codemod {
	return Codemod{
		ID:           "mentions-allow-team-members-to-allowed-collaborators",
		Name:         "Rename allow-team-members to allowed-collaborators in mentions",
		Description:  "Renames allow-team-members to allowed-collaborators in safe-outputs.mentions.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Rename([]string{"safe-outputs", "mentions", "allow-team-members"}, "allowed-collaborators")
			})
		},
	}
}
