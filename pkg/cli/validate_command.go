package cli

import (
	"context"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var validateLog = logger.New("cli:validate_command")

// NewValidateCommand creates the validate command
func NewValidateCommand(validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [workflow]...",
		Short: "Validate agentic workflows without generating lock files",
		Long: `Validate one or more agentic workflows by compiling and running all linters without
generating lock files. This is equivalent to:

  gh aw compile --validate --no-emit --zizmor --actionlint --poutine

If no workflows are specified, all Markdown files in .github/workflows will be validated.

` + WorkflowIDExplanation + `

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` validate                         # Validate all workflows
  ` + string(constants.CLIExtensionPrefix) + ` validate ci-doctor               # Validate a specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` validate ci-doctor daily         # Validate multiple workflows
  ` + string(constants.CLIExtensionPrefix) + ` validate workflow.md             # Validate by file path
  ` + string(constants.CLIExtensionPrefix) + ` validate --dir custom/workflows  # Validate from custom directory
  ` + string(constants.CLIExtensionPrefix) + ` validate --json                  # Output results in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` validate --strict                # Enforce strict mode validation
  ` + string(constants.CLIExtensionPrefix) + ` validate --fail-fast             # Stop at the first error`,
		RunE: func(cmd *cobra.Command, args []string) error {
			engineOverride, _ := cmd.Flags().GetString("engine")
			dir, _ := cmd.Flags().GetString("dir")
			strict, _ := cmd.Flags().GetBool("strict")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			failFast, _ := cmd.Flags().GetBool("fail-fast")
			stats, _ := cmd.Flags().GetBool("stats")
			noCheckUpdate, _ := cmd.Flags().GetBool("no-check-update")
			verbose, _ := cmd.Flags().GetBool("verbose")

			if err := validateEngine(engineOverride); err != nil {
				return err
			}

			// Check for updates (non-blocking, runs once per day)
			CheckForUpdatesAsync(cmd.Context(), noCheckUpdate, verbose)

			validateLog.Printf("Running validate command: workflows=%v, dir=%s", args, dir)

			config := CompileConfig{
				MarkdownFiles:  args,
				Verbose:        verbose,
				EngineOverride: engineOverride,
				Validate:       true,
				NoEmit:         true,
				Zizmor:         true,
				Actionlint:     true,
				Poutine:        true,
				WorkflowDir:    dir,
				Strict:         strict,
				JSONOutput:     jsonOutput,
				FailFast:       failFast,
				Stats:          stats,
			}
			if _, err := CompileWorkflows(context.Background(), config); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringP("engine", "e", "", "Override AI engine (claude, codex, copilot, custom)")
	cmd.Flags().StringP("dir", "d", "", "Workflow directory (default: .github/workflows)")
	cmd.Flags().Bool("strict", false, "Enforce strict mode validation for all workflows")
	cmd.Flags().BoolP("json", "j", false, "Output results in JSON format")
	cmd.Flags().Bool("fail-fast", false, "Stop at the first validation error instead of collecting all errors")
	cmd.Flags().Bool("stats", false, "Display statistics table sorted by file size")
	cmd.Flags().Bool("no-check-update", false, "Skip checking for gh-aw updates")

	// Register completions
	cmd.ValidArgsFunction = CompleteWorkflowNames
	RegisterEngineFlagCompletion(cmd)
	RegisterDirFlagCompletion(cmd, "dir")

	return cmd
}
