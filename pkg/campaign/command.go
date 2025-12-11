package campaign

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// getWorkflowsDir returns the .github/workflows directory path.
// This is a helper to avoid circular dependencies with cli package.
func getWorkflowsDir() string {
	return ".github/workflows"
}

// NewCommand creates the `gh aw campaign` command that surfaces
// first-class campaign definitions from YAML files.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "campaign [filter]",
		Short: "Inspect first-class campaign definitions from campaigns/*.campaign.md",
		Long: `List and inspect first-class campaign definitions declared in YAML files.

Campaigns are defined using Markdown files with YAML frontmatter under the local repository:

	campaigns/*.campaign.md

Each file describes a campaign pattern (ID, name, owners, associated
workflows, repo-memory paths, and risk level). This command provides a
single place to see all campaigns configured for the repo.

Examples:
  ` + constants.CLIExtensionPrefix + ` campaign             # List all campaigns
  ` + constants.CLIExtensionPrefix + ` campaign security    # Filter campaigns by ID or name
  ` + constants.CLIExtensionPrefix + ` campaign --json      # Output campaign definitions as JSON
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var pattern string
			if len(args) > 0 {
				pattern = args[0]
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			return runStatus(pattern, jsonOutput)
		},
	}

	cmd.Flags().Bool("json", false, "Output campaign definitions in JSON format")

	// Subcommand: campaign status
	statusCmd := &cobra.Command{
		Use:   "status [filter]",
		Short: "Show live status for campaigns (compiled workflows, issues, PRs)",
		Long: `Show live status for campaigns, including whether referenced workflows
are compiled and basic issue/PR counts derived from the campaign's
tracker label.

Examples:
  ` + constants.CLIExtensionPrefix + ` campaign status              # Status for all campaigns
  ` + constants.CLIExtensionPrefix + ` campaign status security     # Filter by ID or name
  ` + constants.CLIExtensionPrefix + ` campaign status --json       # JSON status output
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var pattern string
			if len(args) > 0 {
				pattern = args[0]
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			return runRuntimeStatus(pattern, jsonOutput)
		},
	}

	statusCmd.Flags().Bool("json", false, "Output campaign status in JSON format")
	cmd.AddCommand(statusCmd)

	// Subcommand: campaign new
	newCmd := &cobra.Command{
		Use:   "new <id>",
		Short: "Create a new markdown campaign spec under campaigns/",
		Long: `Create a new campaign spec markdown file under campaigns/.

The file will be created as campaigns/<id>.campaign.md with YAML
frontmatter (id, name, version, state, tracker_label) followed by a
markdown body. You can then
update owners, workflows, memory paths, metrics_glob, and governance
fields to match your initiative.

Examples:
  ` + constants.CLIExtensionPrefix + ` campaign new security-q1-2025
  ` + constants.CLIExtensionPrefix + ` campaign new modernization-winter2025 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			force, _ := cmd.Flags().GetBool("force")

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}

			path, err := CreateSpecSkeleton(cwd, id, force)
			if err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created campaign spec at "+path))
			return nil
		},
	}

	newCmd.Flags().Bool("force", false, "Overwrite existing spec file if it already exists")
	cmd.AddCommand(newCmd)

	// Subcommand: campaign validate
	validateCmd := &cobra.Command{
		Use:   "validate [filter]",
		Short: "Validate campaign spec files for common issues",
		Long: `Validate campaign spec files under campaigns/*.campaign.md.

This command performs lightweight semantic validation of campaign
definitions (IDs, tracker labels, workflows, lifecycle state, and
other key fields). By default it exits with a non-zero status when
problems are found.

Examples:
  ` + constants.CLIExtensionPrefix + ` campaign validate              # Validate all campaigns
  ` + constants.CLIExtensionPrefix + ` campaign validate security     # Filter by ID or name
  ` + constants.CLIExtensionPrefix + ` campaign validate --json       # JSON validation report
  ` + constants.CLIExtensionPrefix + ` campaign validate --no-strict  # Report problems without failing`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var pattern string
			if len(args) > 0 {
				pattern = args[0]
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			strict, _ := cmd.Flags().GetBool("strict")
			return runValidate(pattern, jsonOutput, strict)
		},
	}

	validateCmd.Flags().Bool("json", false, "Output campaign validation results in JSON format")
	validateCmd.Flags().Bool("strict", true, "Exit with non-zero status if any problems are found")
	cmd.AddCommand(validateCmd)

	return cmd
}

// runStatus is the implementation for the `gh aw campaign` command.
// It loads campaign specs from the local repository and renders them either
// as a console table or JSON.
func runStatus(pattern string, jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	specs, err := LoadSpecs(cwd)
	if err != nil {
		return err
	}

	specs = FilterSpecs(specs, pattern)

	if jsonOutput {
		jsonBytes, err := json.MarshalIndent(specs, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal campaigns as JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	if len(specs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No campaign specs found. Add files under 'campaigns/*.campaign.md' to define campaigns."))
		return nil
	}

	// Render table to stdout for human-friendly output
	output := console.RenderStruct(specs)
	fmt.Print(output)
	return nil
}

// runRuntimeStatus builds a higher-level view of campaign specs with
// live information derived from GitHub (issue/PR counts) and compiled
// workflow state.
func runRuntimeStatus(pattern string, jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	specs, err := LoadSpecs(cwd)
	if err != nil {
		return err
	}

	specs = FilterSpecs(specs, pattern)
	if len(specs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No campaign specs found. Add files under 'campaigns/*.campaign.md' to define campaigns."))
		return nil
	}

	workflowsDir := getWorkflowsDir()
	var statuses []CampaignRuntimeStatus
	for _, spec := range specs {
		status := BuildRuntimeStatus(spec, workflowsDir)
		statuses = append(statuses, status)
	}

	if jsonOutput {
		jsonBytes, err := json.MarshalIndent(statuses, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal campaign status as JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	output := console.RenderStruct(statuses)
	fmt.Print(output)
	return nil
}

// runValidate loads campaign specs and validates them, returning
// a structured report. When strict is true, the command will exit with
// a non-zero status if any problems are found.
func runValidate(pattern string, jsonOutput bool, strict bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	specs, err := LoadSpecs(cwd)
	if err != nil {
		return err
	}

	specs = FilterSpecs(specs, pattern)
	if len(specs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No campaign specs found. Add files under 'campaigns/*.campaign.md' to define campaigns."))
		return nil
	}

	var results []CampaignValidationResult
	var totalProblems int

	for i := range specs {
		problems := ValidateSpec(&specs[i])
		if len(problems) > 0 {
			log.Printf("Validation problems for campaign '%s' (%s): %v", specs[i].ID, specs[i].ConfigPath, problems)
		}

		results = append(results, CampaignValidationResult{
			ID:         specs[i].ID,
			Name:       specs[i].Name,
			ConfigPath: specs[i].ConfigPath,
			Problems:   problems,
		})
		totalProblems += len(problems)
	}

	if jsonOutput {
		jsonBytes, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal campaign validation results as JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		output := console.RenderStruct(results)
		fmt.Print(output)
	}

	if strict && totalProblems > 0 {
		return fmt.Errorf("campaign validation failed: %d problem(s) found across %d campaign(s)", totalProblems, len(results))
	}

	return nil
}
