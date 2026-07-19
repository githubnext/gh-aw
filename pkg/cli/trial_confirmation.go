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

const trialNestedListIndent = "   "

func formatTrialNestedListItem(item string) string {
	return trialNestedListIndent + console.FormatListItemStderr(item)
}

// trialConfirmationOptions holds parameters for showTrialConfirmation.
type trialConfirmationOptions struct {
	parsedSpecs         []*WorkflowSpec
	logicalRepoSlug     string
	cloneRepoSlug       string
	hostRepoSlug        string
	deleteHostRepo      bool
	forceDeleteHostRepo bool
	autoMergePRs        bool
	repeatCount         int
	directTrialMode     bool
	engineOverride      string
}

// showTrialConfirmation displays a confirmation prompt to the user using parsed workflow specs
func showTrialConfirmation(opts trialConfirmationOptions) error {
	trialConfirmationLog.Printf("Showing trial confirmation: workflows=%d, hostRepo=%s, cloneRepo=%s, repeat=%d, directMode=%v", len(opts.parsedSpecs), opts.hostRepoSlug, opts.cloneRepoSlug, opts.repeatCount, opts.directTrialMode)
	showTrialConfirmationPlan(opts)
	showTrialConfirmationExecutionSteps(opts)
	return showTrialConfirmationPrompt()
}

func showTrialConfirmationPlan(opts trialConfirmationOptions) {
	githubHost := getGitHubHost()
	hostRepoSlugURL := fmt.Sprintf("%s/%s", githubHost, opts.hostRepoSlug)
	var sections []string
	sections = append(sections, console.RenderTitleBox("Trial Execution Plan", 80)...)
	sections = append(sections, "")
	sections = append(sections, console.RenderInfoSection(showTrialConfirmationWorkflowInfo(opts))...)
	sections = append(sections, "")
	sections = append(sections, console.RenderInfoSection(showTrialConfirmationModeInfo(opts))...)
	sections = append(sections, "")
	sections = append(sections, console.RenderInfoSection(fmt.Sprintf("Host Repo:  %s\n            %s", opts.hostRepoSlug, hostRepoSlugURL))...)
	sections = append(sections, "")
	sections = append(sections, console.RenderInfoSection(showTrialConfirmationConfigInfo(opts))...)
	sections = append(sections, "")
	console.RenderComposedSections(sections)
	console.RenderComposedSections(console.RenderTitleBox("Execution Steps", 80))
}

func showTrialConfirmationWorkflowInfo(opts trialConfirmationOptions) string {
	var workflowInfo strings.Builder
	if len(opts.parsedSpecs) == 1 {
		fmt.Fprintf(&workflowInfo, "Workflow:  %s (from %s)", opts.parsedSpecs[0].WorkflowName, opts.parsedSpecs[0].RepoSlug)
	} else {
		workflowInfo.WriteString("Workflows:")
		for _, spec := range opts.parsedSpecs {
			fmt.Fprintf(&workflowInfo, "\n  • %s (from %s)", spec.WorkflowName, spec.RepoSlug)
		}
	}
	return workflowInfo.String()
}

func showTrialConfirmationModeInfo(opts trialConfirmationOptions) string {
	var modeInfo strings.Builder
	if opts.cloneRepoSlug != "" {
		fmt.Fprintf(&modeInfo, "Source:    %s (will be cloned)\n", opts.cloneRepoSlug)
		modeInfo.WriteString("Mode:      Clone repository contents into host repository")
	} else if opts.directTrialMode {
		fmt.Fprintf(&modeInfo, "Target:    %s (direct)\n", opts.hostRepoSlug)
		modeInfo.WriteString("Mode:      Run workflows directly in repository (no simulation)")
	} else {
		fmt.Fprintf(&modeInfo, "Target:    %s (simulated)\n", opts.logicalRepoSlug)
		modeInfo.WriteString("Mode:      Simulate execution against target repository")
	}
	return modeInfo.String()
}

func showTrialConfirmationConfigInfo(opts trialConfirmationOptions) string {
	var configInfo strings.Builder
	if opts.deleteHostRepo {
		configInfo.WriteString("Cleanup:   Host repository will be deleted after completion")
	} else {
		configInfo.WriteString("Cleanup:   Host repository will be preserved")
	}
	if opts.engineOverride != "" {
		fmt.Fprintf(&configInfo, "\nSecrets:   Will prompt for %s API key if needed (stored as repository secret)", opts.engineOverride)
	}
	if opts.repeatCount > 0 {
		fmt.Fprintf(&configInfo, "\nRepeat:    Will run %d times (total executions: %d)", opts.repeatCount, opts.repeatCount+1)
	}
	if opts.autoMergePRs {
		configInfo.WriteString("\nAuto-merge: Pull requests will be automatically merged")
	}
	return configInfo.String()
}

func showTrialConfirmationExecutionSteps(opts trialConfirmationOptions) {
	hostRepoExists := showTrialConfirmationHostRepoExists(opts)
	stepNum := 1
	stepNum = showTrialConfirmationRepoStep(opts, hostRepoExists, stepNum)
	if opts.cloneRepoSlug != "" {
		stepNum = showTrialConfirmationCloneSteps(opts, hostRepoExists, stepNum)
	}
	stepNum = showTrialConfirmationInstallStep(opts, stepNum)
	if opts.engineOverride != "" {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Ensure %s API key secret is configured\n"), stepNum, opts.engineOverride)
		stepNum++
	}
	stepNum = showTrialConfirmationExecuteStep(opts, stepNum)
	if opts.deleteHostRepo {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Delete the host repository\n"), stepNum)
	} else {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Preserve the host repository for inspection\n"), stepNum)
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Fprintln(os.Stderr, "")
}

func showTrialConfirmationHostRepoExists(opts trialConfirmationOptions) bool {
	hostRepoExists := false
	checkCmd := workflow.ExecGH("repo", "view", opts.hostRepoSlug)
	if err := checkCmd.Run(); err == nil {
		hostRepoExists = true
	}
	trialConfirmationLog.Printf("Host repo check: exists=%v, forceDelete=%v", hostRepoExists, opts.forceDeleteHostRepo)
	return hostRepoExists
}

func showTrialConfirmationRepoStep(opts trialConfirmationOptions, hostRepoExists bool, stepNum int) int {
	if hostRepoExists && opts.forceDeleteHostRepo {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Delete and recreate host repository\n"), stepNum)
	} else if hostRepoExists {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Reuse existing host repository\n"), stepNum)
	} else {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Create a private host repository\n"), stepNum)
	}
	return stepNum + 1
}

func showTrialConfirmationCloneSteps(opts trialConfirmationOptions, hostRepoExists bool, stepNum int) int {
	if hostRepoExists && !opts.forceDeleteHostRepo {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Force push contents from %s (overwriting existing content)\n"), stepNum, opts.cloneRepoSlug)
	} else {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Clone contents from %s\n"), stepNum, opts.cloneRepoSlug)
	}
	stepNum++
	if len(opts.parsedSpecs) == 1 {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Disable all workflows in cloned repository except %s\n"), stepNum, opts.parsedSpecs[0].WorkflowName)
	} else {
		workflowNames := sliceutil.Map(opts.parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Disable all workflows in cloned repository except: %s\n"), stepNum, strings.Join(workflowNames, ", "))
	}
	return stepNum + 1
}

func showTrialConfirmationInstallStep(opts trialConfirmationOptions, stepNum int) int {
	if len(opts.parsedSpecs) == 1 {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Install and compile %s\n"), stepNum, opts.parsedSpecs[0].WorkflowName)
	} else {
		workflowNames := sliceutil.Map(opts.parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Install and compile: %s\n"), stepNum, strings.Join(workflowNames, ", "))
	}
	return stepNum + 1
}

func showTrialConfirmationExecuteStep(opts trialConfirmationOptions, stepNum int) int {
	if len(opts.parsedSpecs) == 1 {
		return showTrialConfirmationExecuteSingle(opts, stepNum)
	}
	return showTrialConfirmationExecuteMultiple(opts, stepNum)
}

func showTrialConfirmationExecuteSingle(opts trialConfirmationOptions, stepNum int) int {
	workflowName := opts.parsedSpecs[0].WorkflowName
	if opts.repeatCount > 0 && opts.autoMergePRs {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. For each of %d executions:\n"), stepNum, opts.repeatCount+1)
		fmt.Fprintln(os.Stderr, formatTrialNestedListItem("Execute "+workflowName))
		fmt.Fprintln(os.Stderr, formatTrialNestedListItem("Auto-merge any pull requests created during execution"))
	} else if opts.repeatCount > 0 {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Execute %s %d times\n"), stepNum, workflowName, opts.repeatCount+1)
	} else if opts.autoMergePRs {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Execute %s\n"), stepNum, workflowName)
		stepNum++
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Auto-merge any pull requests created during execution\n"), stepNum)
	} else {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Execute %s\n"), stepNum, workflowName)
	}
	return stepNum + 1
}

func showTrialConfirmationExecuteMultiple(opts trialConfirmationOptions, stepNum int) int {
	workflowNames := sliceutil.Map(opts.parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
	workflowList := strings.Join(workflowNames, ", ")
	if opts.repeatCount > 0 && opts.autoMergePRs {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. For each of %d executions:\n"), stepNum, opts.repeatCount+1)
		fmt.Fprintln(os.Stderr, formatTrialNestedListItem("Execute: "+workflowList))
		fmt.Fprintln(os.Stderr, formatTrialNestedListItem("Auto-merge any pull requests created during execution"))
	} else if opts.repeatCount > 0 {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Execute %d times: %s\n"), stepNum, opts.repeatCount+1, workflowList)
	} else if opts.autoMergePRs {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Execute: %s\n"), stepNum, workflowList)
		stepNum++
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Auto-merge any pull requests created during execution\n"), stepNum)
	} else {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessageStderr("  %d. Execute: %s\n"), stepNum, workflowList)
	}
	return stepNum + 1
}

func showTrialConfirmationPrompt() error {
	confirmed, err := console.ConfirmAction("Do you want to continue?", "Yes, proceed", "No, cancel")
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
