package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/campaign"
	"github.com/githubnext/gh-aw/pkg/console"
)

// validateCampaigns validates campaign spec files and their referenced workflows.
// Returns an error if any campaign specs are invalid or reference missing workflows.
func validateCampaigns(workflowDir string, verbose bool) error {
	// Get absolute path to workflows directory
	absWorkflowDir := workflowDir
	if !filepath.IsAbs(absWorkflowDir) {
		gitRoot, err := findGitRoot()
		if err != nil {
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

	// Load campaign specs
	gitRoot, err := findGitRoot()
	if err != nil {
		// Not in a git repo, can't validate campaigns
		return nil
	}

	specs, err := campaign.LoadSpecs(gitRoot)
	if err != nil {
		return fmt.Errorf("failed to load campaign specs: %w", err)
	}

	if len(specs) == 0 {
		// No campaign specs to validate
		return nil
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Validating %d campaign spec(s)...", len(specs))))
	}

	var allProblems []string
	hasErrors := false

	for _, spec := range specs {
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
		return fmt.Errorf("found %d problem(s) in campaign specs", len(allProblems))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("All %d campaign spec(s) validated successfully", len(specs))))
	}

	return nil
}
