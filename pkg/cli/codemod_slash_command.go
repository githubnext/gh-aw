package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

// getCommandToSlashCommandCodemod migrates on.command to on.slash_command.
func getCommandToSlashCommandCodemod() Codemod {
	return Codemod{
		ID:           "command-to-slash-command-migration",
		Name:         "Migrate on.command to on.slash_command",
		Description:  "Replaces deprecated 'on.command' field with 'on.slash_command'",
		IntroducedIn: "0.2.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Rename([]string{"on", "command"}, "slash_command")
			})
		},
	}
}
