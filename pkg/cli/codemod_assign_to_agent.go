package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

// getAssignToAgentDefaultAgentCodemod migrates safe-outputs.assign-to-agent.default-agent to name.
func getAssignToAgentDefaultAgentCodemod() Codemod {
	return Codemod{
		ID:           "assign-to-agent-default-agent-to-name",
		Name:         "Migrate assign-to-agent default-agent to name",
		Description:  "Renames the deprecated 'default-agent' field to 'name' inside 'safe-outputs.assign-to-agent'",
		IntroducedIn: "0.12.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Rename([]string{"safe-outputs", "assign-to-agent", "default-agent"}, "name")
			})
		},
	}
}
