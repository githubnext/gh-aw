package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// NewSkillCommand creates the skill command.
func NewSkillCommand() *cobra.Command {
	return newSkillCommand(os.Stdout)
}

func newSkillCommand(output io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "skill",
		Short: "Print the agentic-workflows skill",
		Long: `Print the bundled agentic-workflows SKILL.md content to standard output.

Use the skill without initializing a repository or adding files. This command
works outside a Git repository and requires no network access or authentication.`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` skill`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := renderAgenticWorkflowsSkillContent(embeddedFallbackAWMarkdownFiles())
			if err != nil {
				return fmt.Errorf("failed to build agentic-workflows skill: %w", err)
			}
			if _, err := io.WriteString(output, content); err != nil {
				return fmt.Errorf("failed to write agentic-workflows skill: %w", err)
			}
			return nil
		},
		SilenceUsage: true,
	}
}
