package cli

import (
	"errors"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var doctorCommandLog = logger.New("cli:doctor_command")

var runDoctorSetupAuth = RunSetupAuth
var runDoctorSetupRepositoryCheck = RunSetupRepositoryCheck

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
			dir, _ := cmd.Flags().GetString("dir")
			requireOwnerType, _ := cmd.Flags().GetString("require-owner-type")
			verbose, _ := cmd.Flags().GetBool("verbose")

			if repo == "" {
				if cmd.Flags().Changed("dir") || cmd.Flags().Changed("require-owner-type") {
					return errors.New("--dir and --require-owner-type require --repo")
				}

				doctorCommandLog.Print("Running authentication diagnostics (no --repo provided)")
				return runDoctorSetupAuth(SetupAuthOptions{Ctx: cmd.Context(), JSON: jsonOutput})
			}

			doctorCommandLog.Printf("Running repository diagnostics for %q (require-owner-type=%q)", repo, requireOwnerType)
			return runDoctorSetupRepositoryCheck(SetupRepositoryCheckOptions{
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

	return cmd
}
