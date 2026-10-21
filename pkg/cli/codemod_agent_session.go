package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

func getAgentTaskToAgentSessionCodemod() Codemod {
	return Codemod{
		ID:           "agent-task-to-agent-session-migration",
		Name:         "Migrate create-agent-task to create-agent-session",
		Description:  "Replaces deprecated 'safe-outputs.create-agent-task' field with 'safe-outputs.create-agent-session'",
		IntroducedIn: "0.4.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Rename([]string{"safe-outputs", "create-agent-task"}, "create-agent-session")
			})
		},
	}
}
