package workflow

import (
	"fmt"
	"strings"
)

// GitHubScriptStepConfig holds configuration for building a GitHub Script step
type GitHubScriptStepConfig struct {
	// Step metadata
	StepName string // e.g., "Create Output Issue"
	StepID   string // e.g., "create_issue"

	// Main job reference for agent output
	MainJobName string

	// Environment variables specific to this safe output type
	// These are added after GITHUB_AW_AGENT_OUTPUT
	CustomEnvVars []string

	// JavaScript script constant to format and include
	Script string

	// Token configuration (passed to addSafeOutputGitHubTokenForConfig)
	Token string
}

// buildGitHubScriptStep creates a GitHub Script step with common scaffolding
// This extracts the repeated pattern found across safe output job builders
func (c *Compiler) buildGitHubScriptStep(data *WorkflowData, config GitHubScriptStepConfig) []string {
	var steps []string

	// Step name and metadata
	steps = append(steps, fmt.Sprintf("      - name: %s\n", config.StepName))
	steps = append(steps, fmt.Sprintf("        id: %s\n", config.StepID))
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Environment variables section
	steps = append(steps, "        env:\n")

	// Always add the agent output from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", config.MainJobName))

	// Add custom environment variables specific to this safe output type
	steps = append(steps, config.CustomEnvVars...)

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	// With section for github-token
	steps = append(steps, "        with:\n")
	c.addSafeOutputGitHubTokenForConfig(&steps, data, config.Token)
	steps = append(steps, "          script: |\n")

	// Add the formatted JavaScript script
	formattedScript := FormatJavaScriptForYAML(config.Script)
	steps = append(steps, formattedScript...)

	return steps
}

// applySafeOutputEnvToMap adds safe-output related environment variables to an env map
// This extracts the duplicated safe-output env setup logic across all engines (copilot, codex, claude, custom)
func applySafeOutputEnvToMap(env map[string]string, workflowData *WorkflowData) {
	if workflowData.SafeOutputs == nil {
		return
	}

	env["GITHUB_AW_SAFE_OUTPUTS"] = "${{ env.GITHUB_AW_SAFE_OUTPUTS }}"

	// Add staged flag if specified
	if workflowData.TrialMode || workflowData.SafeOutputs.Staged {
		env["GITHUB_AW_SAFE_OUTPUTS_STAGED"] = "true"
	}
	if workflowData.TrialMode && workflowData.TrialTargetRepo != "" {
		env["GITHUB_AW_TARGET_REPO_SLUG"] = workflowData.TrialTargetRepo
	}

	// Add branch name if upload assets is configured
	if workflowData.SafeOutputs.UploadAssets != nil {
		env["GITHUB_AW_ASSETS_BRANCH"] = fmt.Sprintf("%q", workflowData.SafeOutputs.UploadAssets.BranchName)
		env["GITHUB_AW_ASSETS_MAX_SIZE_KB"] = fmt.Sprintf("%d", workflowData.SafeOutputs.UploadAssets.MaxSizeKB)
		env["GITHUB_AW_ASSETS_ALLOWED_EXTS"] = fmt.Sprintf("%q", strings.Join(workflowData.SafeOutputs.UploadAssets.AllowedExts, ","))
	}
}

// applySafeOutputEnvToSlice adds safe-output related environment variables to a YAML string slice
// This is for engines that build YAML line-by-line (like Claude)
func applySafeOutputEnvToSlice(stepLines *[]string, workflowData *WorkflowData) {
	if workflowData.SafeOutputs == nil {
		return
	}

	*stepLines = append(*stepLines, "          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}")

	// Add staged flag if specified
	if workflowData.TrialMode || workflowData.SafeOutputs.Staged {
		*stepLines = append(*stepLines, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"")
	}
	if workflowData.TrialMode && workflowData.TrialTargetRepo != "" {
		*stepLines = append(*stepLines, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q", workflowData.TrialTargetRepo))
	}

	// Add branch name if upload assets is configured
	if workflowData.SafeOutputs.UploadAssets != nil {
		*stepLines = append(*stepLines, fmt.Sprintf("          GITHUB_AW_ASSETS_BRANCH: %q", workflowData.SafeOutputs.UploadAssets.BranchName))
		*stepLines = append(*stepLines, fmt.Sprintf("          GITHUB_AW_ASSETS_MAX_SIZE_KB: %d", workflowData.SafeOutputs.UploadAssets.MaxSizeKB))
		*stepLines = append(*stepLines, fmt.Sprintf("          GITHUB_AW_ASSETS_ALLOWED_EXTS: %q", strings.Join(workflowData.SafeOutputs.UploadAssets.AllowedExts, ",")))
	}
}

// buildSafeOutputJobEnvVars builds environment variables for safe-output jobs with staged/target repo handling
// This extracts the duplicated env setup logic in safe-output job builders (create_issue, add_comment, etc.)
func buildSafeOutputJobEnvVars(trialMode bool, trialLogicalRepoSlug string, staged bool, targetRepoSlug string) []string {
	var customEnvVars []string

	// Pass the staged flag if it's set to true
	if trialMode || staged {
		customEnvVars = append(customEnvVars, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}

	// Set GITHUB_AW_TARGET_REPO_SLUG - prefer target-repo config over trial target repo
	if targetRepoSlug != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q\n", targetRepoSlug))
	} else if trialMode && trialLogicalRepoSlug != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q\n", trialLogicalRepoSlug))
	}

	return customEnvVars
}
