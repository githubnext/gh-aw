package workflow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputsEnvLog = logger.New("workflow:safe_outputs_env")

// ========================================
// Safe Output Environment Variables
// ========================================

// applySafeOutputEnvToMap adds safe-output related environment variables to an env map
// This extracts the duplicated safe-output env setup logic across all engines (copilot, codex, claude, custom)
func applySafeOutputEnvToMap(env map[string]string, data *WorkflowData) {
	if data.SafeOutputs == nil {
		return
	}

	safeOutputsEnvLog.Printf("Applying safe output env vars: trial_mode=%t, staged=%t", data.TrialMode, data.SafeOutputs.Staged)

	env["GH_AW_SAFE_OUTPUTS"] = "${{ steps.set-runtime-paths.outputs.GH_AW_SAFE_OUTPUTS }}"

	// Add staged flag if specified
	if data.TrialMode || data.SafeOutputs.Staged {
		env["GH_AW_SAFE_OUTPUTS_STAGED"] = "true"
	}
	if data.TrialMode && data.TrialLogicalRepo != "" {
		env["GH_AW_TARGET_REPO_SLUG"] = data.TrialLogicalRepo
	}

	// Add branch name if upload assets is configured
	if data.SafeOutputs.UploadAssets != nil {
		safeOutputsEnvLog.Printf("Adding upload assets env vars: branch=%s", data.SafeOutputs.UploadAssets.BranchName)
		env["GH_AW_ASSETS_BRANCH"] = fmt.Sprintf("%q", data.SafeOutputs.UploadAssets.BranchName)
		env["GH_AW_ASSETS_MAX_SIZE_KB"] = strconv.Itoa(data.SafeOutputs.UploadAssets.MaxSizeKB)
		env["GH_AW_ASSETS_ALLOWED_EXTS"] = fmt.Sprintf("%q", strings.Join(data.SafeOutputs.UploadAssets.AllowedExts, ","))
	}
}

// buildWorkflowMetadataEnvVars builds workflow name and source environment variables
// This extracts the duplicated workflow metadata setup logic from safe-output job builders
func buildWorkflowMetadataEnvVars(workflowName string, workflowSource string, localSourceURL string) []string {
	var customEnvVars []string

	// Add workflow name
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))

	// Add workflow source and source URL if present
	if workflowSource != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_SOURCE: %q\n", workflowSource))
		sourceURL := buildSourceURL(workflowSource)
		if sourceURL != "" {
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_SOURCE_URL: %q\n", sourceURL))
		}
	} else if localSourceURL != "" {
		// For local workflows (no external source), use the local file URL so that
		// failure issue links point to the workflow source file rather than "#".
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_SOURCE_URL: %q\n", localSourceURL))
	}

	return customEnvVars
}

// buildWorkflowMetadataEnvVarsWithTrackerID builds workflow metadata env vars including tracker-id
func buildWorkflowMetadataEnvVarsWithTrackerID(workflowName string, workflowSource string, trackerID string, localSourceURL string) []string {
	customEnvVars := buildWorkflowMetadataEnvVars(workflowName, workflowSource, localSourceURL)

	// Add tracker-id if present
	if trackerID != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_TRACKER_ID: %q\n", trackerID))
	}

	return customEnvVars
}

// buildSafeOutputJobEnvVars builds environment variables for safe-output jobs with staged/target repo handling
// This extracts the duplicated env setup logic in safe-output job builders (create_issue, add_comment, etc.)
func buildSafeOutputJobEnvVars(trialMode bool, trialLogicalRepoSlug string, staged bool, targetRepoSlug string) []string {
	var customEnvVars []string

	// Pass the staged flag if it's set to true
	if trialMode || staged {
		safeOutputsEnvLog.Printf("Setting staged flag: trial_mode=%t, staged=%t", trialMode, staged)
		customEnvVars = append(customEnvVars, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}

	// Set GH_AW_TARGET_REPO_SLUG - prefer target-repo config over trial target repo
	if targetRepoSlug != "" {
		safeOutputsEnvLog.Printf("Setting target repo slug from config: %s", targetRepoSlug)
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_TARGET_REPO_SLUG: %q\n", targetRepoSlug))
	} else if trialMode && trialLogicalRepoSlug != "" {
		safeOutputsEnvLog.Printf("Setting target repo slug from trial mode: %s", trialLogicalRepoSlug)
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_TARGET_REPO_SLUG: %q\n", trialLogicalRepoSlug))
	}

	return customEnvVars
}

// buildStandardSafeOutputEnvVars builds the standard set of environment variables
// that all safe-output job builders need: metadata + staged/target repo handling
// This reduces duplication in safe-output job builders
func (c *Compiler) buildStandardSafeOutputEnvVars(data *WorkflowData, targetRepoSlug string) []string {
	var customEnvVars []string

	// Add workflow metadata (name, source, and tracker-id)
	customEnvVars = append(customEnvVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID, buildLocalWorkflowSourceURL(c.markdownPath))...)

	// Add engine metadata (id, version, model) for XML comment marker
	customEnvVars = append(customEnvVars, buildEngineMetadataEnvVars(data.EngineConfig)...)

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		targetRepoSlug,
	)...)

	// Add messages config if present
	if data.SafeOutputs.Messages != nil {
		messagesJSON, err := serializeMessagesConfig(data.SafeOutputs.Messages)
		if err != nil {
			safeOutputsEnvLog.Printf("Warning: failed to serialize messages config: %v", err)
		} else if messagesJSON != "" {
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_MESSAGES: %q\n", messagesJSON))
		}
	}

	return customEnvVars
}

// buildEngineMetadataEnvVars builds engine metadata environment variables (id, version, model)
// These are used by the JavaScript footer generation to create XML comment markers for traceability
func buildEngineMetadataEnvVars(engineConfig *EngineConfig) []string {
	var customEnvVars []string

	if engineConfig == nil {
		return customEnvVars
	}

	safeOutputsEnvLog.Printf("Building engine metadata env vars: id=%s, version=%s", engineConfig.ID, engineConfig.Version)

	// Add engine ID if present
	if engineConfig.ID != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ENGINE_ID: %q\n", engineConfig.ID))
	}

	// Add engine version if present
	if engineConfig.Version != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ENGINE_VERSION: %q\n", engineConfig.Version))
	}

	// Add engine model: prefer explicit compile-time config; fall back to the runtime model
	// captured by the activation job so safe-output footers can show the actual model used
	// (e.g. the value of the GH_AW_MODEL_AGENT_* variable) rather than showing nothing.
	if engineConfig.Model != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ENGINE_MODEL: %q\n", engineConfig.Model))
	} else {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ENGINE_MODEL: ${{ needs.%s.outputs.model }}\n", string(constants.AgentJobName)))
	}

	return customEnvVars
}

// ========================================
// Safe Output Environment Helpers
// ========================================

// addCustomSafeOutputEnvVars adds custom environment variables to safe output job steps
func (c *Compiler) addCustomSafeOutputEnvVars(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs != nil && len(data.SafeOutputs.Env) > 0 {
		for key, value := range data.SafeOutputs.Env {
			*steps = append(*steps, fmt.Sprintf("          %s: %s\n", key, value))
		}
	}
}

func (c *Compiler) addResolvedSafeOutputGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string, resolver func(string) string, allowGitHubApp bool) {
	var safeOutputsToken string
	var githubApp *GitHubAppConfig
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
		githubApp = data.SafeOutputs.GitHubApp
	}

	effectiveCustomToken := configToken
	if effectiveCustomToken == "" {
		effectiveCustomToken = safeOutputsToken
	}

	if allowGitHubApp && githubApp != nil {
		if githubApp.shouldIgnoreMissingKey() {
			fallbackToken := resolver(effectiveCustomToken)
			*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", combineTokenExpressions("${{ steps.safe-outputs-app-token.outputs.token }}", fallbackToken)))
			return
		}
		*steps = append(*steps, "          github-token: ${{ steps.safe-outputs-app-token.outputs.token }}\n")
		return
	}

	effectiveToken := resolver(effectiveCustomToken)
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// addSafeOutputGitHubTokenForConfig adds github-token to the with section for standard safe-output operations.
// Uses precedence:
//   - when safe-outputs.github-app is configured, the app installation token is used
//   - when safe-outputs.github-app ignores missing keys, the app token is primary and the resolved custom token is fallback
//   - otherwise: config token > safe-outputs global github-token > GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
func (c *Compiler) addSafeOutputGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	c.addResolvedSafeOutputGitHubTokenForConfig(steps, data, configToken, getEffectiveSafeOutputGitHubToken, true)
}

// addSafeOutputCopilotGitHubTokenForConfig adds github-token to the with section for Copilot-related operations
// Uses precedence:
//   - when safe-outputs.github-app is configured, the app installation token is used
//   - when safe-outputs.github-app ignores missing keys, the app token is primary and the resolved custom token is fallback
//   - otherwise: config token > safe-outputs global github-token > COPILOT_GITHUB_TOKEN
func (c *Compiler) addSafeOutputCopilotGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	c.addResolvedSafeOutputGitHubTokenForConfig(steps, data, configToken, getEffectiveCopilotRequestsToken, true)
}

// addSafeOutputAgentGitHubTokenForConfig adds github-token to the with section for agent assignment operations
// Uses precedence: config token > safe-outputs token > GH_AW_AGENT_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
// This is specifically for assign-to-agent operations which require elevated permissions.
//
// Note: GitHub App tokens are intentionally NOT used here, even when github-app: is configured.
// The Copilot assignment API only accepts PATs (fine-grained or classic), not GitHub App
// installation tokens. Callers must provide an explicit github-token or rely on GH_AW_AGENT_TOKEN.
func (c *Compiler) addSafeOutputAgentGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	// Get effective token - falls back to ${{ secrets.GH_AW_AGENT_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
	// when no explicit token is provided. GitHub App tokens are never used here because the
	// Copilot assignment API rejects them.
	c.addResolvedSafeOutputGitHubTokenForConfig(steps, data, configToken, getEffectiveCopilotCodingAgentGitHubToken, false)
}

func (c *Compiler) addAllSafeOutputConfigEnvVars(steps *[]string, data *WorkflowData) {
	safeOutputsEnvLog.Print("Adding safe output config environment variables")
	if data.SafeOutputs == nil {
		safeOutputsEnvLog.Print("No safe outputs configured, skipping env var addition")
		return
	}

	// Add the global staged env var once if staged mode is enabled, not in trial mode,
	// and at least one handler is configured. Staged mode is independent of target-repo.
	if !c.trialMode && data.SafeOutputs.Staged && hasAnySafeOutputEnabled(data.SafeOutputs) {
		*steps = append(*steps, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
		safeOutputsEnvLog.Print("Added staged flag")
	}

	// Check if copilot is in create-issue or create-pull-request assignees - enables inline copilot assignment
	if (data.SafeOutputs.CreateIssues != nil && hasCopilotAssignee(data.SafeOutputs.CreateIssues.Assignees)) ||
		(data.SafeOutputs.CreatePullRequests != nil && hasCopilotAssignee(data.SafeOutputs.CreatePullRequests.Assignees)) {
		*steps = append(*steps, "          GH_AW_ASSIGN_COPILOT: \"true\"\n")
		safeOutputsEnvLog.Print("Copilot assignment requested - enabled for create-issue or create-pull-request fallback issues")
	}

	// Note: All handler configuration is read from the config.json file at runtime.
}
