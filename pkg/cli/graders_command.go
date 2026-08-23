package cli

import (
	"errors"
	"strconv"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// NewGradersCommand creates commands for inspecting and replaying workflow graders.
func NewGradersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "graders",
		Short: "Inspect and replay workflow graders",
	}
	cmd.AddCommand(newGradersValueCommand())
	return cmd
}

func newGradersValueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "value <run-id>",
		Short: "Regrade a workflow run's operational value",
		Long: `Regrade the value observation from a completed workflow run at an explicit
evidence cutoff. The command verifies and executes the exact value function archived
by the run. The original artifact is not modified.`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` graders value 123456789 \
    --evidence-at 2026-08-30T12:00:00.000Z --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil || runID <= 0 {
				return errors.New("run ID must be a positive integer")
			}
			evidenceAt, _ := cmd.Flags().GetString("evidence-at")
			repoOverride, _ := cmd.Flags().GetString("repo")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			return RunValueRegrade(cmd.Context(), ValueRegradeConfig{
				RunID:        runID,
				EvidenceAt:   evidenceAt,
				RepoOverride: repoOverride,
				JSONOutput:   jsonOutput,
			})
		},
	}
	cmd.Flags().String("evidence-at", "", "UTC evidence cutoff for this observation")
	_ = cmd.MarkFlagRequired("evidence-at")
	addRepoFlag(cmd)
	addJSONFlag(cmd)
	return cmd
}
