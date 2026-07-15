package cli

import (
	"github.com/spf13/cobra"
)

func NewDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run diagnostics to verify CLI authentication and repository setup",
		Long: `Run diagnostics to verify CLI authentication and repository setup.

Checks GitHub CLI authentication. When --repo is provided, also verifies the
repository exists, resolves the owner type, and inspects checkout state.`,
		Example: `  gh aw doctor
  gh aw doctor --json
  gh aw doctor --repo github/gh-aw
  gh aw doctor --repo github/gh-aw --json
  gh aw doctor --repo github/gh-aw --dir ./gh-aw --require-owner-type org`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, _ := cmd.Flags().GetString("repo")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			if repo == "" {
				return RunSetupAuth(SetupAuthOptions{Ctx: cmd.Context(), JSON: jsonOutput})
			}

			dir, _ := cmd.Flags().GetString("dir")
			requireOwnerType, _ := cmd.Flags().GetString("require-owner-type")

			return RunSetupRepositoryCheck(SetupRepositoryCheckOptions{
				Ctx:              cmd.Context(),
				Repo:             repo,
				Dir:              dir,
				RequireOwnerType: requireOwnerType,
				JSON:             jsonOutput,
			})
		},
	}

	cmd.Flags().StringP("repo", "r", "", "Target repository in owner/repo format")
	cmd.Flags().StringP("dir", "d", "", "Checkout directory to inspect (defaults to the repo name)")
	cmd.Flags().String("require-owner-type", "any", "Require a specific owner type: any, org, or user")
	addJSONFlag(cmd)

	return cmd
}
