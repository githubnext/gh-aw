package campaign

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var campaignLog = logger.New("campaign:command")

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
		Short: "Inspect first-class campaign definitions from .github/workflows/*.campaign.md",
		Long: `List and inspect first-class campaign definitions declared in YAML files.

Campaigns are defined using Markdown files with YAML frontmatter under the local repository:

	.github/workflows/*.campaign.md

Each file describes a campaign pattern (ID, name, owners, associated
workflows, repo-memory paths, and risk level). This command provides a
single place to see all campaigns configured for the repo.

Available subcommands:
	• status   - Show live status for campaigns (compiled workflows, repo-memory)
  • new      - Create a new campaign spec file
  • validate - Validate campaign spec files for common issues

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` campaign                      # List all campaigns
  ` + string(constants.CLIExtensionPrefix) + ` campaign security             # Filter campaigns by ID or name
  ` + string(constants.CLIExtensionPrefix) + ` campaign --json               # Output campaign definitions as JSON
  ` + string(constants.CLIExtensionPrefix) + ` campaign status               # Show live campaign status with issue/PR counts
  ` + string(constants.CLIExtensionPrefix) + ` campaign new security-q1-2025 # Create new campaign spec
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
		Short: "Show live status for campaigns (compiled workflows, repo-memory)",
		Long: `Show live status for campaigns, including whether referenced workflows
are compiled and best-effort campaign metrics derived from repo-memory.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` campaign status              # Status for all campaigns
  ` + string(constants.CLIExtensionPrefix) + ` campaign status security     # Filter by ID or name
  ` + string(constants.CLIExtensionPrefix) + ` campaign status --json       # JSON status output
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
		Use:   "new <campaign-id>",
		Short: "Create a new Markdown campaign spec under .github/workflows/",
		Long: `Create a new campaign spec Markdown file under .github/workflows/.

The file will be created as .github/workflows/<id>.campaign.md with YAML
frontmatter (id, name, version, state, project-url) followed by a
Markdown body. You can then
update owners, workflows, memory paths, metrics-glob, and governance
fields to match your initiative.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` campaign new security-q1-2025
  ` + string(constants.CLIExtensionPrefix) + ` campaign new modernization-winter2025 --force`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				// Build an error message with suggestions but without the leading
				// error prefix icon; the main CLI handler will add that.
				var b strings.Builder
				b.WriteString("missing campaign id argument")
				b.WriteString("\n\nSuggestions:\n")
				suggestions := []string{
					"Provide an ID: '" + string(constants.CLIExtensionPrefix) + " campaign new security-q1-2025'",
					"Use '" + string(constants.CLIExtensionPrefix) + " campaign' to see existing campaigns",
					"Run '" + string(constants.CLIExtensionPrefix) + " help campaign new' for full usage",
				}
				for _, s := range suggestions {
					b.WriteString("  • ")
					b.WriteString(s)
					b.WriteString("\n")
				}

				return errors.New(b.String())
			}

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

			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(
				"Created campaign spec at "+path+". Open this file and fill in owners, workflows, memory-paths, and other details.",
			))
			return nil
		},
	}

	newCmd.Flags().Bool("force", false, "Overwrite existing spec file if it already exists")
	cmd.AddCommand(newCmd)

	// Subcommand: campaign validate
	validateCmd := &cobra.Command{
		Use:   "validate [filter]",
		Short: "Validate campaign spec files for common issues",
		Long: `Validate campaign spec files under .github/workflows/*.campaign.md.

This command performs lightweight semantic validation of campaign
definitions (IDs, workflows, lifecycle state, and
other key fields). By default it exits with a non-zero status when
problems are found.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` campaign validate              # Validate all campaigns
  ` + string(constants.CLIExtensionPrefix) + ` campaign validate security     # Filter by ID or name
  ` + string(constants.CLIExtensionPrefix) + ` campaign validate --json       # JSON validation report
  ` + string(constants.CLIExtensionPrefix) + ` campaign validate --no-strict  # Report problems without failing`,
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
	campaignLog.Printf("Running campaign status with pattern: %s, jsonOutput: %v", pattern, jsonOutput)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	specs, err := LoadSpecs(cwd)
	if err != nil {
		campaignLog.Printf("Failed to load campaign specs: %s", err)
		return err
	}
	campaignLog.Printf("Loaded %d campaign specs", len(specs))

	specs = FilterSpecs(specs, pattern)
	campaignLog.Printf("Filtered to %d campaign specs", len(specs))

	if jsonOutput {
		jsonBytes, err := json.MarshalIndent(specs, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal campaigns as JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	if len(specs) == 0 {
		campaignLog.Print("No campaign specs found matching criteria")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No campaign specs found. Add files under '.github/workflows/*.campaign.md' to define campaigns."))
		return nil
	}

	// Build a compact summary view for human-friendly table output.
	// Full campaign definitions remain available via the --json flag.
	summaries := buildCampaignSummaries(specs)
	output := console.RenderStruct(summaries)
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
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No campaign specs found. Add files under '.github/workflows/*.campaign.md' to define campaigns."))
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
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No campaign specs found. Add files under '.github/workflows/*.campaign.md' to define campaigns."))
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
