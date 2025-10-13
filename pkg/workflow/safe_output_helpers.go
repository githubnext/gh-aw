package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
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

// FilterPermissionErrorPatterns filters error patterns to only those related to permissions
// This helper extracts the common pattern shared by Copilot and Codex engines.
func FilterPermissionErrorPatterns(allPatterns []ErrorPattern) []ErrorPattern {
	var permissionPatterns []ErrorPattern

	for _, pattern := range allPatterns {
		desc := strings.ToLower(pattern.Description)
		if strings.Contains(desc, "permission") ||
			strings.Contains(desc, "unauthorized") ||
			strings.Contains(desc, "forbidden") ||
			strings.Contains(desc, "access") ||
			strings.Contains(desc, "authentication") ||
			strings.Contains(desc, "token") {
			permissionPatterns = append(permissionPatterns, pattern)
		}
	}

	return permissionPatterns
}

// CreateMissingToolEntry creates a missing-tool entry in the safe outputs file
// This helper extracts the common pattern shared by Copilot and Codex engines.
//
// Parameters:
//   - toolName: The name of the tool that encountered a permission error
//   - reason: The reason/error message for the permission denial
//   - verbose: Whether to print verbose output
//
// Returns:
//   - error: An error if the operation failed, nil otherwise
func CreateMissingToolEntry(toolName, reason string, verbose bool) error {
	// Get the safe outputs file path from environment
	safeOutputsFile := os.Getenv("GITHUB_AW_SAFE_OUTPUTS")
	if safeOutputsFile == "" {
		if verbose {
			fmt.Printf("GITHUB_AW_SAFE_OUTPUTS not set, cannot write permission error missing-tool entry\n")
		}
		return nil
	}

	// Create missing-tool entry
	missingToolEntry := map[string]any{
		"type":         "missing-tool",
		"tool":         toolName,
		"reason":       fmt.Sprintf("Permission denied: %s", reason),
		"alternatives": "Check repository permissions and access controls",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	// Convert to JSON and append to safe outputs file
	entryJSON, err := json.Marshal(missingToolEntry)
	if err != nil {
		if verbose {
			fmt.Printf("Failed to marshal missing-tool entry: %v\n", err)
		}
		return err
	}

	// Append to the safe outputs file
	file, err := os.OpenFile(safeOutputsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		if verbose {
			fmt.Printf("Failed to open safe outputs file: %v\n", err)
		}
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(string(entryJSON) + "\n"); err != nil {
		if verbose {
			fmt.Printf("Failed to write missing-tool entry: %v\n", err)
		}
		return err
	}

	if verbose {
		fmt.Printf("Recorded permission error as missing tool: %s\n", toolName)
	}

	return nil
}

// ToolNameExtractor is a function type that extracts tool name from log context
// Used by ScanLogForPermissionErrors to allow engine-specific tool name extraction
type ToolNameExtractor func(lines []string, errorLineIndex int, defaultTool string) string

// ScanLogForPermissionErrors scans log content for permission errors and creates missing-tool entries
// This helper extracts the common pattern shared by Copilot and Codex engines.
//
// Parameters:
//   - logContent: The log content to scan for permission errors
//   - patterns: The permission error patterns to match against
//   - extractToolName: Engine-specific function to extract tool name from context (can be nil)
//   - defaultTool: Default tool name to use if extraction fails
//   - verbose: Whether to print verbose output
func ScanLogForPermissionErrors(
	logContent string,
	patterns []ErrorPattern,
	extractToolName ToolNameExtractor,
	defaultTool string,
	verbose bool,
) {
	lines := strings.Split(logContent, "\n")

	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			continue // Skip invalid patterns
		}

		for i, line := range lines {
			if regex.MatchString(line) {
				// Extract tool name using engine-specific logic or use default
				toolName := defaultTool
				if extractToolName != nil {
					toolName = extractToolName(lines, i, defaultTool)
				}

				// Create missing-tool entry
				if err := CreateMissingToolEntry(toolName, line, verbose); err != nil && verbose {
					fmt.Printf("Warning: failed to create missing-tool entry: %v\n", err)
				}
			}
		}
	}
}
