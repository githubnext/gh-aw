package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var trialConfirmationLog = logger.New("cli:trial_confirmation")

// showTrialConfirmation displays a confirmation prompt to the user using parsed workflow specs
func showTrialConfirmation(parsedSpecs []*WorkflowSpec, logicalRepoSlug, cloneRepoSlug, hostRepoSlug string, deleteHostRepo bool, forceDeleteHostRepo bool, autoMergePRs bool, repeatCount int, directTrialMode bool, engineOverride string) error {
	trialConfirmationLog.Printf("Showing trial confirmation: workflows=%d, hostRepo=%s, cloneRepo=%s, repeat=%d, directMode=%v", len(parsedSpecs), hostRepoSlug, cloneRepoSlug, repeatCount, directTrialMode)
	githubHost := getGitHubHost()
	hostRepoSlugURL := fmt.Sprintf("%s/%s", githubHost, hostRepoSlug)

	var sections []string

	// Title box with double border
	titleText := "Trial Execution Plan"
	sections = append(sections, console.RenderTitleBox(titleText, 80)...)

	sections = append(sections, "")

	// Workflow information section
	var workflowInfo strings.Builder
	if len(parsedSpecs) == 1 {
		fmt.Fprintf(&workflowInfo, "Workflow:  %s (from %s)", parsedSpecs[0].WorkflowName, parsedSpecs[0].RepoSlug)
	} else {
		workflowInfo.WriteString("Workflows:")
		for _, spec := range parsedSpecs {
			fmt.Fprintf(&workflowInfo, "\n  • %s (from %s)", spec.WorkflowName, spec.RepoSlug)
		}
	}

	sections = append(sections, console.RenderInfoSection(workflowInfo.String())...)

	sections = append(sections, "")

	// Display target repository info based on mode
	var modeInfo strings.Builder
	if cloneRepoSlug != "" {
		// Clone-repo mode
		fmt.Fprintf(&modeInfo, "Source:    %s (will be cloned)\n", cloneRepoSlug)
		modeInfo.WriteString("Mode:      Clone repository contents into host repository")
	} else if directTrialMode {
		// Direct trial mode
		fmt.Fprintf(&modeInfo, "Target:    %s (direct)\n", hostRepoSlug)
		modeInfo.WriteString("Mode:      Run workflows directly in repository (no simulation)")
	} else {
		// Logical-repo mode
		fmt.Fprintf(&modeInfo, "Target:    %s (simulated)\n", logicalRepoSlug)
		modeInfo.WriteString("Mode:      Simulate execution against target repository")
	}

	sections = append(sections, console.RenderInfoSection(modeInfo.String())...)

	sections = append(sections, "")

	// Host repository info
	var hostInfo strings.Builder
	fmt.Fprintf(&hostInfo, "Host Repo:  %s\n", hostRepoSlug)
	fmt.Fprintf(&hostInfo, "            %s", hostRepoSlugURL)

	sections = append(sections, console.RenderInfoSection(hostInfo.String())...)

	sections = append(sections, "")

	// Configuration settings
	var configInfo strings.Builder
	if deleteHostRepo {
		configInfo.WriteString("Cleanup:   Host repository will be deleted after completion")
	} else {
		configInfo.WriteString("Cleanup:   Host repository will be preserved")
	}

	// Display secret usage information (only when engine override is specified)
	if engineOverride != "" {
		configInfo.WriteString("\n")
		fmt.Fprintf(&configInfo, "Secrets:   Will prompt for %s API key if needed (stored as repository secret)", engineOverride)
	}

	// Display repeat count if set
	if repeatCount > 0 {
		fmt.Fprintf(&configInfo, "\nRepeat:    Will run %d times (total executions: %d)", repeatCount, repeatCount+1)
	}

	// Display auto-merge setting if enabled
	if autoMergePRs {
		configInfo.WriteString("\nAuto-merge: Pull requests will be automatically merged")
	}

	sections = append(sections, console.RenderInfoSection(configInfo.String())...)

	sections = append(sections, "")

	// Compose and output all sections
	console.RenderComposedSections(sections)

	// Add "Execution Steps" section separator
	executionStepsSections := console.RenderTitleBox("Execution Steps", 80)
	console.RenderComposedSections(executionStepsSections)

	// Check if host repository already exists to update messaging
	hostRepoExists := false
	checkCmd := workflow.ExecGH("repo", "view", hostRepoSlug)
	if err := checkCmd.Run(); err == nil {
		hostRepoExists = true
	}
	trialConfirmationLog.Printf("Host repo check: exists=%v, forceDelete=%v", hostRepoExists, forceDeleteHostRepo)

	// Step 1: Repository creation/reuse
	stepNum := 1
	if hostRepoExists && forceDeleteHostRepo {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Delete and recreate host repository\n"), stepNum)
	} else if hostRepoExists {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Reuse existing host repository\n"), stepNum)
	} else {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Create a private host repository\n"), stepNum)
	}
	stepNum++

	// Step 2: Clone contents (only in clone-repo mode)
	if cloneRepoSlug != "" {
		if hostRepoExists && !forceDeleteHostRepo {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Force push contents from %s (overwriting existing content)\n"), stepNum, cloneRepoSlug)
		} else {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Clone contents from %s\n"), stepNum, cloneRepoSlug)
		}
		stepNum++

		// Show that workflows will be disabled
		if len(parsedSpecs) == 1 {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Disable all workflows in cloned repository except %s\n"), stepNum, parsedSpecs[0].WorkflowName)
		} else {
			workflowNames := sliceutil.Map(parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Disable all workflows in cloned repository except: %s\n"), stepNum, strings.Join(workflowNames, ", "))
		}
		stepNum++
	}

	// Step 3/2: Install and compile workflows
	if len(parsedSpecs) == 1 {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Install and compile %s\n"), stepNum, parsedSpecs[0].WorkflowName)
	} else {
		workflowNames := sliceutil.Map(parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Install and compile: %s\n"), stepNum, strings.Join(workflowNames, ", "))
	}
	stepNum++

	// Step: Configure secrets (only when engine override is specified)
	if engineOverride != "" {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Ensure %s API key secret is configured\n"), stepNum, engineOverride)
		stepNum++
	}

	// Step 5/4: Execute workflows and auto-merge (repeated if --repeat is used)
	if len(parsedSpecs) == 1 {
		workflowName := parsedSpecs[0].WorkflowName
		if repeatCount > 0 && autoMergePRs {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. For each of %d executions:\n"), stepNum, repeatCount+1)
			fmt.Fprintf(os.Stderr, "     a. Execute %s\n", workflowName)
			fmt.Fprintf(os.Stderr, "     b. Auto-merge any pull requests created during execution\n")
		} else if repeatCount > 0 {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Execute %s %d times\n"), stepNum, workflowName, repeatCount+1)
		} else if autoMergePRs {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Execute %s\n"), stepNum, workflowName)
			stepNum++
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Auto-merge any pull requests created during execution\n"), stepNum)
		} else {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Execute %s\n"), stepNum, workflowName)
		}
	} else {
		workflowNames := sliceutil.Map(parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
		workflowList := strings.Join(workflowNames, ", ")

		if repeatCount > 0 && autoMergePRs {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. For each of %d executions:\n"), stepNum, repeatCount+1)
			fmt.Fprintf(os.Stderr, "     a. Execute: %s\n", workflowList)
			fmt.Fprintf(os.Stderr, "     b. Auto-merge any pull requests created during execution\n")
		} else if repeatCount > 0 {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Execute %d times: %s\n"), stepNum, repeatCount+1, workflowList)
		} else if autoMergePRs {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Execute: %s\n"), stepNum, workflowList)
			stepNum++
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Auto-merge any pull requests created during execution\n"), stepNum)
		} else {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Execute: %s\n"), stepNum, workflowList)
		}
	}
	stepNum++

	// Final step: Delete/preserve repository
	if deleteHostRepo {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Delete the host repository\n"), stepNum)
	} else {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  %d. Preserve the host repository for inspection\n"), stepNum)
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Fprintln(os.Stderr, "")

	// Ask for confirmation using console helper
	confirmed, err := console.ConfirmAction(
		"Do you want to continue?",
		"Yes, proceed",
		"No, cancel",
	)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}

	if !confirmed {
		trialConfirmationLog.Print("Trial cancelled by user")
		return errors.New("trial cancelled by user")
	}

	trialConfirmationLog.Print("Trial confirmed by user, proceeding")
	return nil
}
