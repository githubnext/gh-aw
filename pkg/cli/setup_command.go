package cli

import (
	"github.com/spf13/cobra"
)

func NewSetupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "setup",
		Short:  "Run reusable auth and repository setup checks",
		Hidden: true,
		Long: `Run reusable auth and repository setup checks.

This command exposes the shared setup primitives that bootstrap and future setup
flows can reuse, without forcing a full bootstrap run.

Available subcommands:
  - auth - Verify GitHub CLI authentication
  - repo - Check repository existence, owner type, and checkout state`,
		Example: `  gh aw setup auth
  gh aw setup repo --repo github/gh-aw
  gh aw setup repo --repo github/gh-aw --json
  gh aw setup repo --repo github/gh-aw --dir ./gh-aw --require-owner-type org`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newSetupAuthSubcommand())
	cmd.AddCommand(newSetupRepoSubcommand())
	return cmd
}

func newSetupAuthSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Verify GitHub CLI authentication",
		Long: `Verify that GitHub CLI authentication is available for follow-on setup tasks.

This reuses the same authentication preflight used by bootstrap and other
setup-oriented commands.`,
		Example: `  gh aw setup auth
  gh aw setup auth --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			return RunSetupAuth(SetupAuthOptions{Ctx: cmd.Context(), JSON: jsonOutput})
		},
	}
	addJSONFlag(cmd)
	return cmd
}

func newSetupRepoSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Check repository and checkout setup state",
		Long: `Check repository and checkout setup state.

This command verifies GitHub CLI authentication, confirms that the target
repository exists, resolves the owner type, and inspects whether the target
directory is already attached to the expected checkout or is ready for clone.`,
		Example: `  gh aw setup repo --repo github/gh-aw
  gh aw setup repo --repo github/gh-aw --json
  gh aw setup repo --repo github/gh-aw --dir ./gh-aw
  gh aw setup repo --repo github/gh-aw --require-owner-type org`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, _ := cmd.Flags().GetString("repo")
			dir, _ := cmd.Flags().GetString("dir")
			requireOwnerType, _ := cmd.Flags().GetString("require-owner-type")
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			return RunSetupRepositoryCheck(SetupRepositoryCheckOptions{
				Ctx:              cmd.Context(),
				Repo:             repo,
				Dir:              dir,
				RequireOwnerType: requireOwnerType,
				Verbose:          verbose,
				JSON:             jsonOutput,
			})
		},
	}

	cmd.Flags().StringP("repo", "r", "", "Target repository in owner/repo format")
	cmd.Flags().StringP("dir", "d", "", "Checkout directory to inspect (defaults to the repo name)")
	cmd.Flags().String("require-owner-type", "any", "Require a specific owner type: any, org, or user")
	addJSONFlag(cmd)
	_ = cmd.MarkFlagRequired("repo")

	return cmd
}
