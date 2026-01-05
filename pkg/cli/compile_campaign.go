// Package cli provides campaign workflow compilation and validation.
//
// This file handles validation of campaign spec files and their referenced workflows.
// Campaign workflows are special workflows that orchestrate multiple sub-workflows
// across a set of target repositories.
//
// # Organization Rationale
//
// The validateCampaigns function is co-located with campaign compilation logic because:
//   - It's domain-specific to campaign workflows
//   - It's only called during compile operations
//   - It's tightly coupled to the campaign compilation process
//   - The file is small (98 lines) and focused
//
// This follows the principle that domain-specific validation belongs in domain files.
// See skills/developer/SKILL.md for validation architecture patterns.
//
// # Validation Functions
//
// Campaign Validation:
//   - validateCampaigns() - Validates campaign specs and referenced workflows
//
// This validation ensures that:
//   - Campaign spec files are syntactically valid
//   - Referenced workflow files exist in the workflows directory
//   - Campaign configuration is complete and correct
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/campaign"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var compileCampaignLog = logger.New("cli:compile_campaign")

// validateCampaigns validates campaign spec files and their referenced workflows.
// If campaignFiles is provided (non-nil), only those specific campaign files are validated.
// If campaignFiles is nil, all campaign specs are validated.
// Returns an error if any campaign specs are invalid or reference missing workflows.
func validateCampaigns(workflowDir string, verbose bool, campaignFiles []string) error {
	compileCampaignLog.Printf("Validating campaigns with workflow directory: %s", workflowDir)

	// Get absolute path to workflows directory
	absWorkflowDir := workflowDir
	if !filepath.IsAbs(absWorkflowDir) {
		gitRoot, err := findGitRoot()
		if err != nil {
			compileCampaignLog.Print("Not in a git repository, using current directory")
			// If not in a git repo, use current directory
			cwd, cwdErr := os.Getwd()
			if cwdErr != nil {
				return nil // Silently skip if we can't determine the directory
			}
			absWorkflowDir = filepath.Join(cwd, workflowDir)
		} else {
			absWorkflowDir = filepath.Join(gitRoot, workflowDir)
		}
	}
	compileCampaignLog.Printf("Using absolute workflow directory: %s", absWorkflowDir)

	// Load campaign specs
	gitRoot, err := findGitRoot()
	if err != nil {
		compileCampaignLog.Print("Cannot validate campaigns: not in a git repository")
		// Not in a git repo, can't validate campaigns
		return nil
	}

	specs, err := campaign.LoadSpecs(gitRoot)
	if err != nil {
		compileCampaignLog.Printf("Failed to load campaign specs: %v", err)
		return fmt.Errorf("failed to load campaign specs: %w", err)
	}

	if len(specs) == 0 {
		compileCampaignLog.Print("No campaign specs found to validate")
		// No campaign specs to validate
		return nil
	}

	// Filter specs if specific campaign files were provided
	var specsToValidate []campaign.CampaignSpec
	if campaignFiles != nil && len(campaignFiles) > 0 {
		compileCampaignLog.Printf("Filtering to validate only %d specific campaign file(s)", len(campaignFiles))
		// Create a map of absolute paths for quick lookup
		campaignFileMap := make(map[string]bool)
		for _, cf := range campaignFiles {
			absPath, err := filepath.Abs(cf)
			if err == nil {
				campaignFileMap[absPath] = true
			}
		}

		for _, spec := range specs {
			// Get absolute path of the spec's config file
			specPath := spec.ConfigPath
			if !filepath.IsAbs(specPath) {
				specPath = filepath.Join(gitRoot, specPath)
			}
			absSpecPath, err := filepath.Abs(specPath)
			if err == nil && campaignFileMap[absSpecPath] {
				specsToValidate = append(specsToValidate, spec)
			}
		}
		compileCampaignLog.Printf("Filtered to %d campaign spec(s) for validation", len(specsToValidate))
	} else {
		// Validate all specs
		specsToValidate = specs
		compileCampaignLog.Printf("Loaded %d campaign specs for validation", len(specs))
	}

	if len(specsToValidate) == 0 {
		compileCampaignLog.Print("No matching campaign specs found to validate")
		return nil
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Validating %d campaign spec(s)...", len(specsToValidate))))
	}

	var allProblems []string
	hasErrors := false

	for _, spec := range specsToValidate {
		// Validate the spec itself
		problems := campaign.ValidateSpec(&spec)

		// Validate that referenced workflows exist
		workflowProblems := campaign.ValidateWorkflowsExist(&spec, absWorkflowDir)
		problems = append(problems, workflowProblems...)

		if len(problems) > 0 {
			hasErrors = true
			for _, problem := range problems {
				msg := fmt.Sprintf("Campaign '%s' (%s): %s", spec.ID, spec.ConfigPath, problem)
				allProblems = append(allProblems, msg)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(msg))
				}
			}
		}
	}

	if hasErrors {
		compileCampaignLog.Printf("Campaign validation completed with %d problems", len(allProblems))
		return fmt.Errorf("found %d problem(s) in campaign specs", len(allProblems))
	}

	compileCampaignLog.Printf("All %d campaign specs validated successfully", len(specsToValidate))
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("All %d campaign spec(s) validated successfully", len(specsToValidate))))
	}

	return nil
}
