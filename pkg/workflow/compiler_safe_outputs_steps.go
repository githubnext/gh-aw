package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var consolidatedSafeOutputsStepsLog = logger.New("workflow:compiler_safe_outputs_steps")

// buildConsolidatedSafeOutputStep builds a single step for a safe output operation
// within the consolidated safe-outputs job. This function handles both inline script
// mode and file mode (requiring from local filesystem).
func (c *Compiler) buildConsolidatedSafeOutputStep(data *WorkflowData, config SafeOutputStepConfig) []string {
	var steps []string

	// Build step condition if provided
	var conditionStr string
	if config.Condition != nil {
		conditionStr = config.Condition.Render()
	}

	// Step name and metadata
	steps = append(steps, fmt.Sprintf("      - name: %s\n", config.StepName))
	steps = append(steps, fmt.Sprintf("        id: %s\n", config.StepID))
	if conditionStr != "" {
		steps = append(steps, fmt.Sprintf("        if: %s\n", conditionStr))
	}
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

	// Environment variables section
	steps = append(steps, "        env:\n")
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}\n")
	steps = append(steps, config.CustomEnvVars...)

	// Add custom safe output env vars
	c.addCustomSafeOutputEnvVars(&steps, data)

	// With section for github-token
	steps = append(steps, "        with:\n")
	if config.UseCopilotCodingAgentToken {
		c.addSafeOutputAgentGitHubTokenForConfig(&steps, data, config.Token)
	} else if config.UseCopilotRequestsToken {
		c.addSafeOutputCopilotGitHubTokenForConfig(&steps, data, config.Token)
	} else {
		c.addSafeOutputGitHubTokenForConfig(&steps, data, config.Token)
	}

	steps = append(steps, "          script: |\n")

	// Add the formatted JavaScript script
	// Use require mode if ScriptName is set, otherwise inline the bundled script
	if config.ScriptName != "" {
		// Require mode: Use setup_globals helper
		steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
		steps = append(steps, "            setupGlobals(core, github, context, exec, io);\n")
		steps = append(steps, fmt.Sprintf("            const { main } = require('"+SetupActionDestination+"/%s.cjs');\n", config.ScriptName))
		steps = append(steps, "            await main();\n")
	} else {
		// Inline JavaScript: Use setup_globals helper
		steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
		steps = append(steps, "            setupGlobals(core, github, context, exec, io);\n")
		// Inline mode: embed the bundled script directly
		formattedScript := FormatJavaScriptForYAML(config.Script)
		steps = append(steps, formattedScript...)
	}

	return steps
}

// buildSharedPRCheckoutSteps builds checkout and git configuration steps that are shared
// between create-pull-request and push-to-pull-request-branch operations.
// These steps are added once with a combined condition to avoid duplication.
func (c *Compiler) buildSharedPRCheckoutSteps(data *WorkflowData) []string {
	consolidatedSafeOutputsStepsLog.Print("Building shared PR checkout steps")
	var steps []string

	// Determine which token to use for checkout
	var checkoutToken string
	var gitRemoteToken string
	if data.SafeOutputs.App != nil {
		// nolint:gosec // G101: False positive - this is a GitHub Actions expression template placeholder, not a hardcoded credential
		checkoutToken = "${{ steps.safe-outputs-app-token.outputs.token }}" //nolint:gosec
		// nolint:gosec // G101: False positive - this is a GitHub Actions expression template placeholder, not a hardcoded credential
		gitRemoteToken = "${{ steps.safe-outputs-app-token.outputs.token }}"
	} else {
		// Use token precedence chain instead of hardcoded github.token
		// Precedence: create-pull-request config token > push-to-pull-request-branch config token > safe-outputs token > GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
		var createPRToken string
		if data.SafeOutputs.CreatePullRequests != nil {
			createPRToken = data.SafeOutputs.CreatePullRequests.GitHubToken
		}
		var pushToPRBranchToken string
		if data.SafeOutputs.PushToPullRequestBranch != nil {
			pushToPRBranchToken = data.SafeOutputs.PushToPullRequestBranch.GitHubToken
		}
		var safeOutputsToken string
		if data.SafeOutputs != nil {
			safeOutputsToken = data.SafeOutputs.GitHubToken
		}
		// Choose the first non-empty custom token for precedence
		// Priority: create-pull-request token > push-to-pull-request-branch token > safe-outputs token
		effectiveCustomToken := createPRToken
		if effectiveCustomToken == "" {
			effectiveCustomToken = pushToPRBranchToken
		}
		if effectiveCustomToken == "" {
			effectiveCustomToken = safeOutputsToken
		}
		// Get effective token (handles fallback to GH_AW_GITHUB_TOKEN || GITHUB_TOKEN)
		effectiveToken := getEffectiveSafeOutputGitHubToken(effectiveCustomToken)
		// nolint:gosec // G101: False positive - this is a GitHub Actions expression template placeholder, not a hardcoded credential
		checkoutToken = effectiveToken
		// nolint:gosec // G101: False positive - this is a GitHub Actions expression template placeholder, not a hardcoded credential
		gitRemoteToken = effectiveToken
	}

	// Build combined condition: execute if either create_pull_request or push_to_pull_request_branch will run
	var condition ConditionNode
	if data.SafeOutputs.CreatePullRequests != nil && data.SafeOutputs.PushToPullRequestBranch != nil {
		// Both enabled: combine conditions with OR
		condition = BuildOr(
			BuildSafeOutputType("create_pull_request"),
			BuildSafeOutputType("push_to_pull_request_branch"),
		)
	} else if data.SafeOutputs.CreatePullRequests != nil {
		// Only create_pull_request
		condition = BuildSafeOutputType("create_pull_request")
	} else {
		// Only push_to_pull_request_branch
		condition = BuildSafeOutputType("push_to_pull_request_branch")
	}

	// Determine target repository for checkout and git config
	// Priority: create-pull-request target-repo > trialLogicalRepoSlug > default (source repo)
	var targetRepoSlug string
	if data.SafeOutputs.CreatePullRequests != nil && data.SafeOutputs.CreatePullRequests.TargetRepoSlug != "" {
		targetRepoSlug = data.SafeOutputs.CreatePullRequests.TargetRepoSlug
		consolidatedSafeOutputsStepsLog.Printf("Using target-repo from create-pull-request: %s", targetRepoSlug)
	} else if c.trialMode && c.trialLogicalRepoSlug != "" {
		targetRepoSlug = c.trialLogicalRepoSlug
		consolidatedSafeOutputsStepsLog.Printf("Using trialLogicalRepoSlug: %s", targetRepoSlug)
	}

	// Determine the ref (branch) to checkout
	// Priority: create-pull-request base-branch > default to github.ref_name
	// This is critical: we must checkout the base branch, not github.sha (the triggering commit),
	// because github.sha might be an older commit with different workflow files. A shallow clone
	// of an old commit followed by git fetch/checkout may not properly update all files,
	// leading to spurious "workflow file changed" errors on push.
	var checkoutRef string
	if data.SafeOutputs.CreatePullRequests != nil && data.SafeOutputs.CreatePullRequests.BaseBranch != "" {
		checkoutRef = data.SafeOutputs.CreatePullRequests.BaseBranch
		consolidatedSafeOutputsStepsLog.Printf("Using base-branch from create-pull-request for checkout ref: %s", checkoutRef)
	} else {
		// Default to github.ref_name which is the branch name that triggered the workflow
		checkoutRef = "${{ github.ref_name }}"
		consolidatedSafeOutputsStepsLog.Print("Using github.ref_name for checkout ref")
	}

	// Step 1: Checkout repository with conditional execution
	steps = append(steps, "      - name: Checkout repository\n")
	steps = append(steps, fmt.Sprintf("        if: %s\n", condition.Render()))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/checkout")))
	steps = append(steps, "        with:\n")

	// Set repository parameter if checking out a different repository
	if targetRepoSlug != "" {
		steps = append(steps, fmt.Sprintf("          repository: %s\n", targetRepoSlug))
		consolidatedSafeOutputsStepsLog.Printf("Added repository parameter: %s", targetRepoSlug)
	}

	// Set ref to checkout the base branch, not github.sha
	steps = append(steps, fmt.Sprintf("          ref: %s\n", checkoutRef))
	steps = append(steps, fmt.Sprintf("          token: %s\n", checkoutToken))
	steps = append(steps, "          persist-credentials: false\n")
	steps = append(steps, "          fetch-depth: 1\n")

	// Step 2: Configure Git credentials with conditional execution
	// Security: Pass GitHub token through environment variable to prevent template injection

	// Determine REPO_NAME value based on target repository
	repoNameValue := "${{ github.repository }}"
	if targetRepoSlug != "" {
		repoNameValue = fmt.Sprintf("%q", targetRepoSlug)
		consolidatedSafeOutputsStepsLog.Printf("Using target repo for REPO_NAME: %s", targetRepoSlug)
	}

	gitConfigSteps := []string{
		"      - name: Configure Git credentials\n",
		fmt.Sprintf("        if: %s\n", condition.Render()),
		"        env:\n",
		fmt.Sprintf("          REPO_NAME: %s\n", repoNameValue),
		"          SERVER_URL: ${{ github.server_url }}\n",
		fmt.Sprintf("          GIT_TOKEN: %s\n", gitRemoteToken),
		"        run: |\n",
		"          git config --global user.email \"github-actions[bot]@users.noreply.github.com\"\n",
		"          git config --global user.name \"github-actions[bot]\"\n",
		"          # Re-authenticate git with GitHub token\n",
		"          SERVER_URL_STRIPPED=\"${SERVER_URL#https://}\"\n",
		"          git remote set-url origin \"https://x-access-token:${GIT_TOKEN}@${SERVER_URL_STRIPPED}/${REPO_NAME}.git\"\n",
		"          echo \"Git configured with standard GitHub Actions identity\"\n",
	}
	steps = append(steps, gitConfigSteps...)

	consolidatedSafeOutputsStepsLog.Printf("Added shared checkout with condition: %s", condition.Render())
	return steps
}

// buildHandlerManagerStep builds a single step that uses the safe output handler manager
// to dispatch messages to appropriate handlers. This replaces multiple individual steps
// with a single dispatcher step that processes all safe output types.
func (c *Compiler) buildHandlerManagerStep(data *WorkflowData) []string {
	consolidatedSafeOutputsStepsLog.Print("Building handler manager step")

	var steps []string

	// Step name and metadata
	steps = append(steps, "      - name: Process Safe Outputs\n")
	steps = append(steps, "        id: process_safe_outputs\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

	// Environment variables
	steps = append(steps, "        env:\n")
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}\n")

	// Note: The project handler manager has been removed.
	// All project-related operations are now handled by the unified handler.

	// Add custom safe output env vars
	c.addCustomSafeOutputEnvVars(&steps, data)

	// Add handler manager config as JSON
	c.addHandlerManagerConfigEnvVar(&steps, data)

	// Add all safe output configuration env vars (still needed by individual handlers)
	c.addAllSafeOutputConfigEnvVars(&steps, data)

	// Add GH_AW_PROJECT_URL and GH_AW_PROJECT_GITHUB_TOKEN environment variables for project operations
	// These are set from the project URL and token configured in any project-related safe-output:
	// - update-project
	// - create-project-status-update
	// - create-project
	//
	// The project field is REQUIRED in update-project and create-project-status-update (enforced by schema validation)
	// Agents can optionally override this per-message by including a project field in their output
	//
	// Note: If multiple project configs are present, we prefer update-project > create-project-status-update > create-project
	// This is only relevant for the environment variables - each configuration must explicitly specify its own settings
	var projectURL string
	var projectToken string

	// Check update-project first (highest priority)
	if data.SafeOutputs.UpdateProjects != nil && data.SafeOutputs.UpdateProjects.Project != "" {
		projectURL = data.SafeOutputs.UpdateProjects.Project
		// Use per-config token, fallback to safe-outputs level token, then default
		configToken := data.SafeOutputs.UpdateProjects.GitHubToken
		if configToken == "" && data.SafeOutputs.GitHubToken != "" {
			configToken = data.SafeOutputs.GitHubToken
		}
		projectToken = getEffectiveProjectGitHubToken(configToken)
		consolidatedSafeOutputsStepsLog.Printf("Setting GH_AW_PROJECT_URL from update-project config: %s", projectURL)
		consolidatedSafeOutputsStepsLog.Printf("Setting GH_AW_PROJECT_GITHUB_TOKEN from update-project config")
	} else if data.SafeOutputs.CreateProjectStatusUpdates != nil && data.SafeOutputs.CreateProjectStatusUpdates.Project != "" {
		projectURL = data.SafeOutputs.CreateProjectStatusUpdates.Project
		// Use per-config token, fallback to safe-outputs level token, then default
		configToken := data.SafeOutputs.CreateProjectStatusUpdates.GitHubToken
		if configToken == "" && data.SafeOutputs.GitHubToken != "" {
			configToken = data.SafeOutputs.GitHubToken
		}
		projectToken = getEffectiveProjectGitHubToken(configToken)
		consolidatedSafeOutputsStepsLog.Printf("Setting GH_AW_PROJECT_URL from create-project-status-update config: %s", projectURL)
		consolidatedSafeOutputsStepsLog.Printf("Setting GH_AW_PROJECT_GITHUB_TOKEN from create-project-status-update config")
	}

	// Check create-project for token even if no URL is set (create-project doesn't have a project URL field)
	// This ensures GH_AW_PROJECT_GITHUB_TOKEN is set when create-project is configured
	if projectToken == "" && data.SafeOutputs.CreateProjects != nil {
		// Use per-config token, fallback to safe-outputs level token, then default
		configToken := data.SafeOutputs.CreateProjects.GitHubToken
		if configToken == "" && data.SafeOutputs.GitHubToken != "" {
			configToken = data.SafeOutputs.GitHubToken
		}
		projectToken = getEffectiveProjectGitHubToken(configToken)
		consolidatedSafeOutputsStepsLog.Printf("Setting GH_AW_PROJECT_GITHUB_TOKEN from create-project config")
	}

	if projectURL != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_PROJECT_URL: %q\n", projectURL))
	}

	if projectToken != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_PROJECT_GITHUB_TOKEN: %s\n", projectToken))
	}

	// With section for github-token
	// Use the standard safe outputs token for all operations.
	// If project operations are configured, prefer the project token for the github-script client.
	// Rationale: update_project/create_project_status_update call the Projects v2 GraphQL API, which
	// cannot be accessed with the default GITHUB_TOKEN. GH_AW_PROJECT_GITHUB_TOKEN is the required
	// token for Projects v2 operations.
	steps = append(steps, "        with:\n")
	configToken := ""
	if projectToken != "" {
		configToken = projectToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, configToken)

	steps = append(steps, "          script: |\n")
	steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
	steps = append(steps, "            setupGlobals(core, github, context, exec, io);\n")
	steps = append(steps, "            const { main } = require('"+SetupActionDestination+"/safe_output_handler_manager.cjs');\n")
	steps = append(steps, "            await main();\n")

	return steps
}
