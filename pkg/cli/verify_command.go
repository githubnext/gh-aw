package cli

import (
	"context"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var verifyLog = logger.New("cli:verify_command")

// NewVerifyCommand creates the verify command
func NewVerifyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify [workflow]...",
		Short: "Verify agentic workflows without generating lock files",
		Long: `Verify one or more agentic workflows by compiling and running all linters without
generating lock files. This is equivalent to:

  gh aw compile --validate --no-emit --zizmor --actionlint --poutine

If no workflows are specified, all Markdown files in .github/workflows will be verified.

` + WorkflowIDExplanation + `

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` verify                    # Verify all workflows
  ` + string(constants.CLIExtensionPrefix) + ` verify ci-doctor          # Verify a specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` verify ci-doctor daily    # Verify multiple workflows
  ` + string(constants.CLIExtensionPrefix) + ` verify workflow.md        # Verify by file path
  ` + string(constants.CLIExtensionPrefix) + ` verify --dir custom/workflows  # Verify from custom directory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			verbose, _ := cmd.Flags().GetBool("verbose")

			verifyLog.Printf("Running verify command: workflows=%v, dir=%s", args, dir)

			config := CompileConfig{
				MarkdownFiles: args,
				Verbose:       verbose,
				Validate:      true,
				NoEmit:        true,
				Zizmor:        true,
				Actionlint:    true,
				Poutine:       true,
				WorkflowDir:   dir,
			}
			if _, err := CompileWorkflows(context.Background(), config); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringP("dir", "d", "", "Workflow directory (default: .github/workflows)")

	// Register completions
	cmd.ValidArgsFunction = CompleteWorkflowNames
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}
