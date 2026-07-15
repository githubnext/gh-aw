package cli

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

func NewBootstrapCommand(validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "bootstrap [source]...",
		Short:  "Bootstrap a repository for agentic workflows",
		Hidden: true,
		Long: `Bootstrap a repository for agentic workflows by combining repository setup,
checkout attachment or cloning, initialization, and optional workflow or package installation.

This command is intentionally generic. It handles repository creation and initialization
without assuming any product-specific auth, app registration, or secret layout.

When you pass one or more sources, bootstrap will add them after initialization and then
compile workflows unless you disable compilation with --no-compile.

Sources use the same syntax as '` + string(constants.CLIExtensionPrefix) + ` add'.`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` bootstrap --repo octo-org/platform-ops --create-repo --visibility private
  ` + string(constants.CLIExtensionPrefix) + ` bootstrap --repo octo-org/platform-ops github/central-agentic-ops/readiness
  ` + string(constants.CLIExtensionPrefix) + ` bootstrap --repo octo-org/platform-ops ./local-workflow.md --yes
  ` + string(constants.CLIExtensionPrefix) + ` bootstrap --repo octo-org/platform-ops --plan
  ` + string(constants.CLIExtensionPrefix) + ` bootstrap --repo octo-org/platform-ops --engine claude --create-repo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			engineOverride, _ := cmd.Flags().GetString("engine")
			if err := validateEngine(engineOverride); err != nil {
				return err
			}

			repo, _ := cmd.Flags().GetString("repo")
			dir, _ := cmd.Flags().GetString("dir")
			createRepo, _ := cmd.Flags().GetBool("create-repo")
			visibility, _ := cmd.Flags().GetString("visibility")
			requireOwnerType, _ := cmd.Flags().GetString("require-owner-type")
			yes, _ := cmd.Flags().GetBool("yes")
			planOnly, _ := cmd.Flags().GetBool("plan")
			force, _ := cmd.Flags().GetBool("force")
			noCompile, _ := cmd.Flags().GetBool("no-compile")
			verbose, _ := cmd.Flags().GetBool("verbose")

			if repo == "" {
				return fmt.Errorf("--repo is required. Example: %s --repo github/gh-aw\n\nRun '%s --help' for usage information", cmd.CommandPath(), cmd.CommandPath())
			}

			return RunBootstrap(BootstrapOptions{
				Ctx:              cmd.Context(),
				Repo:             repo,
				Dir:              dir,
				CreateRepo:       createRepo,
				Visibility:       visibility,
				RequireOwnerType: requireOwnerType,
				Yes:              yes,
				PlanOnly:         planOnly,
				EngineOverride:   engineOverride,
				Sources:          args,
				Force:            force,
				NoCompile:        noCompile,
				Verbose:          verbose,
			})
		},
	}

	cmd.Flags().String("repo", "", "Target repository (OWNER/REPO format)")
	cmd.Flags().String("dir", "", "Local checkout directory (defaults to the repository name)")
	cmd.Flags().Bool("create-repo", false, "Create the target repository when it does not exist")
	cmd.Flags().String("visibility", "private", "Repository visibility for --create-repo: private, public, or internal")
	cmd.Flags().String("require-owner-type", "any", "Require the repository owner to be org, user, or any")
	cmd.Flags().BoolP("yes", "y", false, "Apply the bootstrap plan without confirmation")
	cmd.Flags().Bool("plan", false, "Print the bootstrap plan without making changes")
	cmd.Flags().BoolP("force", "f", false, "Allow added workflows to overwrite existing files when the add step runs")
	cmd.Flags().Bool("no-compile", false, "Skip workflow compilation after adding sources")
	addEngineFlag(cmd)
	RegisterEngineFlagCompletion(cmd)

	return cmd
}
