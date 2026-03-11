package workflow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputsJobsLog = logger.New("workflow:safe_outputs_jobs")

// ========================================
// Safe Output Job Configuration and Builder
// ========================================

// SafeOutputJobConfig holds configuration for building a safe output job
// This config struct extracts the common parameters across all safe output job builders
type SafeOutputJobConfig struct {
	// Job metadata
	JobName     string // e.g., "create_issue"
	StepName    string // e.g., "Create Output Issue"
	StepID      string // e.g., "create_issue"
	MainJobName string // Main workflow job name for dependencies

	// Custom environment variables specific to this safe output type
	CustomEnvVars []string

	// JavaScript script constant to include in the GitHub Script step
	Script string

	// Script name for looking up custom action path (optional)
	// If provided and action mode is custom, the compiler will use a custom action
	// instead of inline JavaScript. Example: "create_issue"
	ScriptName string

	// Job configuration
	Permissions                *Permissions      // Job permissions
	Outputs                    map[string]string // Job outputs
	Condition                  ConditionNode     // Job condition (if clause)
	Needs                      []string          // Job dependencies
	PreSteps                   []string          // Optional steps to run before the GitHub Script step
	PostSteps                  []string          // Optional steps to run after the GitHub Script step
	Token                      string            // GitHub token for this output type
	UseCopilotRequestsToken    bool              // Whether to use Copilot token preference chain
	UseCopilotCodingAgentToken bool              // Whether to use agent token preference chain (config token > GH_AW_AGENT_TOKEN)
	TargetRepoSlug             string            // Target repository for cross-repo operations
}

// buildSafeOutputJob creates a safe output job with common scaffolding
// This extracts the repeated pattern found across safe output job builders:
// 1. Validate configuration
// 2. Build custom environment variables
// 3. Invoke buildGitHubScriptStep
// 4. Create Job with standard metadata
func (c *Compiler) buildSafeOutputJob(data *WorkflowData, config SafeOutputJobConfig) (*Job, error) {
	safeOutputsJobsLog.Printf("Building safe output job: %s (actionMode=%s)", config.JobName, c.actionMode)
	var steps []string

	// Add GitHub App token minting step if app is configured
	if data.SafeOutputs != nil && data.SafeOutputs.GitHubApp != nil {
		safeOutputsJobsLog.Print("Adding GitHub App token minting step with auto-computed permissions")
		steps = append(steps, c.buildGitHubAppTokenMintStep(data.SafeOutputs.GitHubApp, config.Permissions)...)
	}

	// Add pre-steps if provided (e.g., checkout, git config for create-pull-request)
	if len(config.PreSteps) > 0 {
		safeOutputsJobsLog.Printf("Adding %d pre-steps to job", len(config.PreSteps))
		steps = append(steps, config.PreSteps...)
	}

	// Build the step based on action mode
	var scriptSteps []string
	if c.actionMode.UsesExternalActions() && config.ScriptName != "" {
		// Use custom action mode (dev or release) if enabled and script name is provided
		safeOutputsJobsLog.Printf("Using custom action mode (%s) for script: %s", c.actionMode, config.ScriptName)
		scriptSteps = c.buildCustomActionStep(data, GitHubScriptStepConfig{
			StepName:                   config.StepName,
			StepID:                     config.StepID,
			MainJobName:                config.MainJobName,
			CustomEnvVars:              config.CustomEnvVars,
			Script:                     config.Script,
			CustomToken:                config.Token,
			UseCopilotRequestsToken:    config.UseCopilotRequestsToken,
			UseCopilotCodingAgentToken: config.UseCopilotCodingAgentToken,
		}, config.ScriptName)
	} else {
		// Use inline mode (default behavior)
		// If ScriptName is provided, convert it to ScriptFile (.cjs extension)
		scriptFile := ""
		if config.ScriptName != "" {
			scriptFile = config.ScriptName + ".cjs"
			safeOutputsJobsLog.Printf("Using inline mode with external script: %s", scriptFile)
		} else {
			safeOutputsJobsLog.Printf("Using inline mode (actions/github-script)")
		}
		scriptSteps = c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
			StepName:                   config.StepName,
			StepID:                     config.StepID,
			MainJobName:                config.MainJobName,
			CustomEnvVars:              config.CustomEnvVars,
			Script:                     config.Script,
			ScriptFile:                 scriptFile,
			CustomToken:                config.Token,
			UseCopilotRequestsToken:    config.UseCopilotRequestsToken,
			UseCopilotCodingAgentToken: config.UseCopilotCodingAgentToken,
		})
	}
	steps = append(steps, scriptSteps...)

	// Add post-steps if provided (e.g., assignees, reviewers)
	if len(config.PostSteps) > 0 {
		steps = append(steps, config.PostSteps...)
	}

	// Add GitHub App token invalidation step if app is configured
	if data.SafeOutputs != nil && data.SafeOutputs.GitHubApp != nil {
		safeOutputsJobsLog.Print("Adding GitHub App token invalidation step")
		steps = append(steps, c.buildGitHubAppTokenInvalidationStep()...)
	}

	// Determine job condition
	jobCondition := config.Condition
	if jobCondition == nil {
		safeOutputsJobsLog.Printf("No custom condition provided, using default for job: %s", config.JobName)
		jobCondition = BuildSafeOutputType(config.JobName)
	}

	// Determine job needs
	needs := config.Needs
	if len(needs) == 0 {
		needs = []string{config.MainJobName}
	}
	safeOutputsJobsLog.Printf("Job %s needs: %v", config.JobName, needs)

	// Create the job with standard configuration
	job := &Job{
		Name:           config.JobName,
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Environment:    c.indentYAMLLines(resolveSafeOutputsEnvironment(data), "    "),
		Permissions:    config.Permissions.RenderToYAML(),
		TimeoutMinutes: 10, // 10-minute timeout as required for all safe output jobs
		Steps:          steps,
		Outputs:        config.Outputs,
		Needs:          needs,
	}

	return job, nil
}

var safeOutputsStepsLog = logger.New("workflow:safe_outputs_steps")

// ========================================
// Safe Output Step Builders
// ========================================

// buildCustomActionStep creates a step that uses a custom action reference
// instead of inline JavaScript via actions/github-script
func (c *Compiler) buildCustomActionStep(data *WorkflowData, config GitHubScriptStepConfig, scriptName string) []string {
	safeOutputsStepsLog.Printf("Building custom action step: %s (scriptName=%s, actionMode=%s)", config.StepName, scriptName, c.actionMode)

	var steps []string

	// Get the action path from the script registry
	actionPath := DefaultScriptRegistry.GetActionPath(scriptName)
	if actionPath == "" {
		safeOutputsStepsLog.Printf("WARNING: No action path found for script %s, falling back to inline mode", scriptName)
		// Set ScriptFile for inline mode fallback
		config.ScriptFile = scriptName + ".cjs"
		return c.buildGitHubScriptStep(data, config)
	}

	// Resolve the action reference based on mode
	actionRef := c.resolveActionReference(actionPath, data)
	if actionRef == "" {
		safeOutputsStepsLog.Printf("WARNING: Could not resolve action reference for %s, falling back to inline mode", actionPath)
		// Set ScriptFile for inline mode fallback
		config.ScriptFile = scriptName + ".cjs"
		return c.buildGitHubScriptStep(data, config)
	}

	// Add artifact download steps before the custom action step
	steps = append(steps, buildAgentOutputDownloadSteps()...)

	// Step name and metadata
	steps = append(steps, fmt.Sprintf("      - name: %s\n", config.StepName))
	steps = append(steps, fmt.Sprintf("        id: %s\n", config.StepID))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", actionRef))

	// Environment variables section
	steps = append(steps, "        env:\n")
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}\n")
	steps = append(steps, config.CustomEnvVars...)
	c.addCustomSafeOutputEnvVars(&steps, data)

	// With section for inputs (replaces github-token in actions/github-script)
	steps = append(steps, "        with:\n")

	// Map github-token to token input for custom actions
	c.addCustomActionGitHubToken(&steps, data, config)

	return steps
}

// addCustomActionGitHubToken adds a GitHub token as action input.
// The token precedence depends on the tokenType flags:
// - UseCopilotCodingAgentToken: customToken > SafeOutputs.GitHubToken > GH_AW_AGENT_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
// - UseCopilotRequestsToken: customToken > SafeOutputs.GitHubToken > COPILOT_GITHUB_TOKEN
// - Default: customToken > SafeOutputs.GitHubToken > GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
func (c *Compiler) addCustomActionGitHubToken(steps *[]string, data *WorkflowData, config GitHubScriptStepConfig) {
	var token string

	// Get safe-outputs level token
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}

	// Choose the first non-empty custom token for precedence
	effectiveCustomToken := config.CustomToken
	if effectiveCustomToken == "" {
		effectiveCustomToken = safeOutputsToken
	}

	// Agent token mode: use full precedence chain for agent assignment
	if config.UseCopilotCodingAgentToken {
		token = getEffectiveCopilotCodingAgentGitHubToken(effectiveCustomToken)
	} else if config.UseCopilotRequestsToken {
		// Copilot mode: use getEffectiveCopilotRequestsToken with safe-outputs token precedence
		token = getEffectiveCopilotRequestsToken(effectiveCustomToken)
	} else {
		// Standard mode: use safe output token chain
		token = getEffectiveSafeOutputGitHubToken(effectiveCustomToken)
	}

	*steps = append(*steps, fmt.Sprintf("          token: %s\n", token))
}

// GitHubScriptStepConfig holds configuration for building a GitHub Script step
type GitHubScriptStepConfig struct {
	// Step metadata
	StepName string // e.g., "Create Output Issue"
	StepID   string // e.g., "create_issue"

	// Main job reference for agent output
	MainJobName string

	// Environment variables specific to this safe output type
	// These are added after GH_AW_AGENT_OUTPUT
	CustomEnvVars []string

	// JavaScript script constant to format and include (for inline mode)
	Script string

	// ScriptFile is the .cjs filename to require (e.g., "noop.cjs")
	// If empty, Script will be inlined instead
	ScriptFile string

	// CustomToken configuration (passed to addSafeOutputGitHubTokenForConfig or addSafeOutputCopilotGitHubTokenForConfig)
	CustomToken string

	// UseCopilotRequestsToken indicates whether to use the Copilot token preference chain
	// custom token > COPILOT_GITHUB_TOKEN
	// This should be true for Copilot-related operations like creating agent tasks,
	// assigning copilot to issues, or adding copilot as PR reviewer
	UseCopilotRequestsToken bool

	// UseCopilotCodingAgentToken indicates whether to use the agent token preference chain
	// (config token > GH_AW_AGENT_TOKEN)
	// This should be true for agent assignment operations (assign-to-agent)
	UseCopilotCodingAgentToken bool
}

// buildGitHubScriptStep creates a GitHub Script step with common scaffolding
// This extracts the repeated pattern found across safe output job builders
func (c *Compiler) buildGitHubScriptStep(data *WorkflowData, config GitHubScriptStepConfig) []string {
	safeOutputsStepsLog.Printf("Building GitHub Script step: %s (useCopilotRequestsToken=%v, useCopilotCodingAgentToken=%v)", config.StepName, config.UseCopilotRequestsToken, config.UseCopilotCodingAgentToken)

	var steps []string

	// Add artifact download steps before the GitHub Script step
	steps = append(steps, buildAgentOutputDownloadSteps()...)

	// Step name and metadata
	steps = append(steps, fmt.Sprintf("      - name: %s\n", config.StepName))
	steps = append(steps, fmt.Sprintf("        id: %s\n", config.StepID))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

	// Environment variables section
	steps = append(steps, "        env:\n")

	// Read GH_AW_AGENT_OUTPUT from environment (set by artifact download step)
	// instead of directly from job outputs which may be masked by GitHub Actions
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}\n")

	// Add custom environment variables specific to this safe output type
	steps = append(steps, config.CustomEnvVars...)

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	// With section for github-token
	steps = append(steps, "        with:\n")
	if config.UseCopilotCodingAgentToken {
		c.addSafeOutputAgentGitHubTokenForConfig(&steps, data, config.CustomToken)
	} else if config.UseCopilotRequestsToken {
		c.addSafeOutputCopilotGitHubTokenForConfig(&steps, data, config.CustomToken)
	} else {
		c.addSafeOutputGitHubTokenForConfig(&steps, data, config.CustomToken)
	}

	steps = append(steps, "          script: |\n")

	// Use require() if ScriptFile is specified, otherwise inline the script
	if config.ScriptFile != "" {
		steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
		steps = append(steps, "            setupGlobals(core, github, context, exec, io);\n")
		steps = append(steps, fmt.Sprintf("            const { main } = require('"+SetupActionDestination+"/%s');\n", config.ScriptFile))
		steps = append(steps, "            await main();\n")
	} else {
		// Add the formatted JavaScript script (inline)
		formattedScript := FormatJavaScriptForYAML(config.Script)
		steps = append(steps, formattedScript...)
	}

	return steps
}

// buildGitHubScriptStepWithoutDownload creates a GitHub Script step without artifact download steps
// This is useful when multiple script steps are needed in the same job and artifact downloads
// should only happen once at the beginning
func (c *Compiler) buildGitHubScriptStepWithoutDownload(data *WorkflowData, config GitHubScriptStepConfig) []string {
	safeOutputsStepsLog.Printf("Building GitHub Script step without download: %s", config.StepName)

	var steps []string

	// Step name and metadata (no artifact download steps)
	steps = append(steps, fmt.Sprintf("      - name: %s\n", config.StepName))
	steps = append(steps, fmt.Sprintf("        id: %s\n", config.StepID))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

	// Environment variables section
	steps = append(steps, "        env:\n")

	// Read GH_AW_AGENT_OUTPUT from environment (set by artifact download step)
	// instead of directly from job outputs which may be masked by GitHub Actions
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}\n")

	// Add custom environment variables specific to this safe output type
	steps = append(steps, config.CustomEnvVars...)

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	// With section for github-token
	steps = append(steps, "        with:\n")
	if config.UseCopilotCodingAgentToken {
		c.addSafeOutputAgentGitHubTokenForConfig(&steps, data, config.CustomToken)
	} else if config.UseCopilotRequestsToken {
		c.addSafeOutputCopilotGitHubTokenForConfig(&steps, data, config.CustomToken)
	} else {
		c.addSafeOutputGitHubTokenForConfig(&steps, data, config.CustomToken)
	}

	steps = append(steps, "          script: |\n")

	// Use require() if ScriptFile is specified, otherwise inline the script
	if config.ScriptFile != "" {
		steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
		steps = append(steps, "            setupGlobals(core, github, context, exec, io);\n")
		steps = append(steps, fmt.Sprintf("            const { main } = require('"+SetupActionDestination+"/%s');\n", config.ScriptFile))
		steps = append(steps, "            await main();\n")
	} else {
		// Add the formatted JavaScript script (inline)
		formattedScript := FormatJavaScriptForYAML(config.Script)
		steps = append(steps, formattedScript...)
	}

	return steps
}

// buildAgentOutputDownloadSteps creates steps to download the agent output artifact
// and set the GH_AW_AGENT_OUTPUT environment variable for safe-output jobs.
// GH_AW_AGENT_OUTPUT is only set when the artifact was actually downloaded successfully.
func buildAgentOutputDownloadSteps() []string {
	return buildArtifactDownloadSteps(ArtifactDownloadConfig{
		ArtifactName:     constants.AgentArtifactName,   // Unified agent artifact
		ArtifactFilename: constants.AgentOutputFilename, // Filename inside the artifact directory
		DownloadPath:     "/tmp/gh-aw/",
		SetupEnvStep:     true,
		EnvVarName:       "GH_AW_AGENT_OUTPUT",
		StepName:         "Download agent output artifact",
		StepID:           "download-agent-output",
	})
}

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

	env["GH_AW_SAFE_OUTPUTS"] = "${{ env.GH_AW_SAFE_OUTPUTS }}"

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

// applySafeOutputEnvToSlice adds safe-output related environment variables to a YAML string slice
// This is for engines that build YAML line-by-line (like Claude)
func applySafeOutputEnvToSlice(stepLines *[]string, workflowData *WorkflowData) {
	if workflowData.SafeOutputs == nil {
		return
	}

	*stepLines = append(*stepLines, "          GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}")

	// Add staged flag if specified
	if workflowData.TrialMode || workflowData.SafeOutputs.Staged {
		*stepLines = append(*stepLines, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"")
	}
	if workflowData.TrialMode && workflowData.TrialLogicalRepo != "" {
		*stepLines = append(*stepLines, fmt.Sprintf("          GH_AW_TARGET_REPO_SLUG: %q", workflowData.TrialLogicalRepo))
	}

	// Add branch name if upload assets is configured
	if workflowData.SafeOutputs.UploadAssets != nil {
		*stepLines = append(*stepLines, fmt.Sprintf("          GH_AW_ASSETS_BRANCH: %q", workflowData.SafeOutputs.UploadAssets.BranchName))
		*stepLines = append(*stepLines, fmt.Sprintf("          GH_AW_ASSETS_MAX_SIZE_KB: %d", workflowData.SafeOutputs.UploadAssets.MaxSizeKB))
		*stepLines = append(*stepLines, fmt.Sprintf("          GH_AW_ASSETS_ALLOWED_EXTS: %q", strings.Join(workflowData.SafeOutputs.UploadAssets.AllowedExts, ",")))
	}
}

// buildWorkflowMetadataEnvVars builds workflow name and source environment variables
// This extracts the duplicated workflow metadata setup logic from safe-output job builders
func buildWorkflowMetadataEnvVars(workflowName string, workflowSource string) []string {
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
	}

	return customEnvVars
}

// buildWorkflowMetadataEnvVarsWithTrackerID builds workflow metadata env vars including tracker-id
func buildWorkflowMetadataEnvVarsWithTrackerID(workflowName string, workflowSource string, trackerID string) []string {
	customEnvVars := buildWorkflowMetadataEnvVars(workflowName, workflowSource)

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
	customEnvVars = append(customEnvVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID)...)

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

// buildStepLevelSafeOutputEnvVars builds environment variables for consolidated safe output steps
// This excludes variables that are already set at the job level in consolidated jobs
func (c *Compiler) buildStepLevelSafeOutputEnvVars(data *WorkflowData, targetRepoSlug string) []string {
	var customEnvVars []string

	// Only add target repo slug if it's different from the job-level setting
	// (i.e., this step has a specific target-repo config that overrides the global trial mode target)
	if targetRepoSlug != "" {
		// Step-specific target repo overrides job-level setting
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_TARGET_REPO_SLUG: %q\n", targetRepoSlug))
	} else if !c.trialMode && data.SafeOutputs.Staged {
		// Step needs staged flag but there's no job-level target repo (not in trial mode)
		// Job level only sets this if trialMode is true
		customEnvVars = append(customEnvVars, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}

	// Note: The following are now set at job level and should NOT be included here:
	// - GH_AW_WORKFLOW_NAME
	// - GH_AW_WORKFLOW_SOURCE
	// - GH_AW_WORKFLOW_SOURCE_URL
	// - GH_AW_TRACKER_ID
	// - GH_AW_ENGINE_ID
	// - GH_AW_ENGINE_VERSION
	// - GH_AW_ENGINE_MODEL
	// - GH_AW_SAFE_OUTPUTS_STAGED (if in trial mode)
	// - GH_AW_TARGET_REPO_SLUG (if in trial mode and no step override)
	// - GH_AW_SAFE_OUTPUT_MESSAGES

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

	// Add engine model if present
	if engineConfig.Model != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ENGINE_MODEL: %q\n", engineConfig.Model))
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

// addSafeOutputGitHubTokenForConfig adds github-token to the with section, preferring per-config token over global
// Uses precedence: config token > safe-outputs global github-token > GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
func (c *Compiler) addSafeOutputGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}

	// If app is configured, use app token
	if data.SafeOutputs != nil && data.SafeOutputs.GitHubApp != nil {
		*steps = append(*steps, "          github-token: ${{ steps.safe-outputs-app-token.outputs.token }}\n")
		return
	}

	// Choose the first non-empty custom token for precedence
	effectiveCustomToken := configToken
	if effectiveCustomToken == "" {
		effectiveCustomToken = safeOutputsToken
	}

	// Get effective token
	effectiveToken := getEffectiveSafeOutputGitHubToken(effectiveCustomToken)
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// addSafeOutputCopilotGitHubTokenForConfig adds github-token to the with section for Copilot-related operations
// Uses precedence: config token > safe-outputs global github-token > COPILOT_GITHUB_TOKEN
func (c *Compiler) addSafeOutputCopilotGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}

	// If app is configured, use app token
	if data.SafeOutputs != nil && data.SafeOutputs.GitHubApp != nil {
		*steps = append(*steps, "          github-token: ${{ steps.safe-outputs-app-token.outputs.token }}\n")
		return
	}

	// Choose the first non-empty custom token for precedence
	effectiveCustomToken := configToken
	if effectiveCustomToken == "" {
		effectiveCustomToken = safeOutputsToken
	}

	// Get effective token
	effectiveToken := getEffectiveCopilotRequestsToken(effectiveCustomToken)
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// addSafeOutputAgentGitHubTokenForConfig adds github-token to the with section for agent assignment operations
// Uses precedence: config token > safe-outputs token > GH_AW_AGENT_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
// This is specifically for assign-to-agent operations which require elevated permissions.
//
// Note: GitHub App tokens are intentionally NOT used here, even when github-app: is configured.
// The Copilot assignment API only accepts PATs (fine-grained or classic), not GitHub App
// installation tokens. Callers must provide an explicit github-token or rely on GH_AW_AGENT_TOKEN.
func (c *Compiler) addSafeOutputAgentGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	// Get safe-outputs level token
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}

	// Choose the first non-empty custom token for precedence
	effectiveCustomToken := configToken
	if effectiveCustomToken == "" {
		effectiveCustomToken = safeOutputsToken
	}

	// Get effective token - falls back to ${{ secrets.GH_AW_AGENT_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
	// when no explicit token is provided. GitHub App tokens are never used here because the
	// Copilot assignment API rejects them.
	effectiveToken := getEffectiveCopilotCodingAgentGitHubToken(effectiveCustomToken)
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// buildTitlePrefixEnvVar builds a title-prefix environment variable line for safe-output jobs.
// envVarName should be the full env var name like "GH_AW_ISSUE_TITLE_PREFIX" or "GH_AW_DISCUSSION_TITLE_PREFIX".
// Returns an empty slice if titlePrefix is empty.
func buildTitlePrefixEnvVar(envVarName string, titlePrefix string) []string {
	if titlePrefix == "" {
		return nil
	}
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, titlePrefix)}
}

// buildLabelsEnvVar builds a labels environment variable line for safe-output jobs.
// envVarName should be the full env var name like "GH_AW_ISSUE_LABELS" or "GH_AW_PR_LABELS".
// Returns an empty slice if labels is empty.
func buildLabelsEnvVar(envVarName string, labels []string) []string {
	if len(labels) == 0 {
		return nil
	}
	labelsStr := strings.Join(labels, ",")
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, labelsStr)}
}

// buildCategoryEnvVar builds a category environment variable line for discussion safe-output jobs.
// envVarName should be the full env var name like "GH_AW_DISCUSSION_CATEGORY".
// Returns an empty slice if category is empty.
func buildCategoryEnvVar(envVarName string, category string) []string {
	if category == "" {
		return nil
	}
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, category)}
}

// buildAllowedReposEnvVar builds an allowed-repos environment variable line for safe-output jobs.
// envVarName should be the full env var name like "GH_AW_ALLOWED_REPOS".
// Returns an empty slice if allowedRepos is empty.
func buildAllowedReposEnvVar(envVarName string, allowedRepos []string) []string {
	if len(allowedRepos) == 0 {
		return nil
	}
	reposStr := strings.Join(allowedRepos, ",")
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, reposStr)}
}
