package cli

import (
	"github.com/github/gh-aw/pkg/constants"
	impactrunner "github.com/github/gh-aw/pkg/impactscore/runner"
	"github.com/spf13/cobra"
)

// NewImpactCommand creates the hidden impact score command.
func NewImpactCommand() *cobra.Command {
	cfg := impactrunner.DefaultConfig()
	cmd := &cobra.Command{
		Use:    "impact",
		Short:  "Score repository work and workflow impact (experimental)",
		Hidden: true,
		Long: `[EXPERIMENTAL] Score repository work items and attribute impact to agentic workflows.

This hidden command reads repository work from GitHub, discovers agentic workflow lock files,
loads the repo-local impact policy from .github/workflows/aw.json when present, and writes
JSON artifacts plus a text or HTML report.`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` impact
  ` + string(constants.CLIExtensionPrefix) + ` impact --report-format html
	` + string(constants.CLIExtensionPrefix) + ` impact --out ./impact-score`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := impactrunner.Run(cmd.Context(), cfg)
			return err
		},
	}

	cmd.Flags().StringVar(&cfg.OutDir, "out", cfg.OutDir, "Output directory for JSON artifacts")
	cmd.Flags().IntVar(&cfg.MaxItems, "max-items", cfg.MaxItems, "Maximum issues/PRs to fetch")
	cmd.Flags().StringVar(&cfg.State, "state", cfg.State, "GitHub issue/PR state filter: open, closed, or all; close reason and merge status are scored separately")
	cmd.Flags().StringVar(&cfg.ReportFormat, "report-format", cfg.ReportFormat, "Report rendering: text or html")

	_ = cmd.RegisterFlagCompletionFunc("state", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"open", "closed", "all"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("report-format", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "html"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
