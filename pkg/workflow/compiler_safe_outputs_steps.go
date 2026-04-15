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
		conditionStr = RenderCondition(config.Condition)
	}

	// Step name and metadata
	steps = append(steps, fmt.Sprintf("      - name: %s\n", config.StepName))
	steps = append(steps, fmt.Sprintf("        id: %s\n", config.StepID))
	if conditionStr != "" {
		steps = append(steps, fmt.Sprintf("        if: %s\n", conditionStr))
	}
	if config.ContinueOnError {
		steps = append(steps, "        continue-on-error: true\n")
	}
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))

	// Environment variables section
	steps = append(steps, "        env:\n")
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ steps.setup-agent-output-env.outputs.GH_AW_AGENT_OUTPUT }}\n")
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
		steps = append(steps, "            setupGlobals(core, github, context, exec, io, getOctokit);\n")
		steps = append(steps, fmt.Sprintf("            const { main } = require('"+SetupActionDestination+"/%s.cjs');\n", config.ScriptName))
		steps = append(steps, "            await main();\n")
	} else {
		// Inline JavaScript: Use setup_globals helper
		steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
		steps = append(steps, "            setupGlobals(core, github, context, exec, io, getOctokit);\n")
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
	// Uses computeEffectivePRCheckoutToken for consistent token resolution (GitHub App or PAT chain)
	checkoutToken, _ := computeEffectivePRCheckoutToken(data.SafeOutputs)
	gitRemoteToken := checkoutToken

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
	// Priority: create-pull-request base-branch > fallback expression
	// This is critical: we must checkout the base branch, not github.sha (the triggering commit),
	// because github.sha might be an older commit with different workflow files. A shallow clone
	// of an old commit followed by git fetch/checkout may not properly update all files,
	// leading to spurious "workflow file changed" errors on push.
	//
	// Fallback expression: github.base_ref || github.event.pull_request.base.ref || github.ref_name || github.event.repository.default_branch
	// - github.base_ref: set for pull_request/pull_request_target events
	// - github.event.pull_request.base.ref: set for pull_request_review, pull_request_review_comment events
	// - github.event.repository.default_branch: fallback for issue_comment events and other edge cases
	//
	// LIMITATION: For issue_comment events on PRs targeting non-default branches, this will checkout
	// the default branch instead of the actual PR base branch. This is a known limitation because
	// issue_comment payloads don't include PR base ref info and we can't make API calls in YAML expressions.
	// For most PRs targeting main/master, this works correctly.
	//
	// TODO: @dsyme says: We must remove this. Indeed the important longer term thing is that we need the processing
	// of the application of safe outputs to be independent of
	// * event trigger context
	// * ideally repository context too
	// So safe outputs are "self-describing" and already know which base branch, repository etc. they're
	// targeting.  Then a lot of this gnarly event code will be only on the "front end" (prepping the
	// coding agent) not the "backend" (applying the safe outputs)
	const baseBranchFallbackExpr = "${{ github.base_ref || github.event.pull_request.base.ref || github.ref_name || github.event.repository.default_branch }}"
	var checkoutRef string
	if data.SafeOutputs.CreatePullRequests != nil && data.SafeOutputs.CreatePullRequests.BaseBranch != "" {
		checkoutRef = data.SafeOutputs.CreatePullRequests.BaseBranch
		consolidatedSafeOutputsStepsLog.Printf("Using custom base-branch from create-pull-request for checkout ref: %s", checkoutRef)
	} else {
		checkoutRef = baseBranchFallbackExpr
		consolidatedSafeOutputsStepsLog.Printf("Using fallback base branch expression for checkout ref")
	}

	// Step 1: Checkout repository with conditional execution
	steps = append(steps, "      - name: Checkout repository\n")
	steps = append(steps, fmt.Sprintf("        if: %s\n", RenderCondition(condition)))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getActionPin("actions/checkout")))
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
		fmt.Sprintf("        if: %s\n", RenderCondition(condition)),
		"        env:\n",
		fmt.Sprintf("          REPO_NAME: %s\n", repoNameValue),
		"          SERVER_URL: ${{ github.server_url }}\n",
		fmt.Sprintf("          GIT_TOKEN: %s\n", gitRemoteToken),
		"        run: |\n",
		"          git config --global user.email \"github-actions[bot]@users.noreply.github.com\"\n",
		"          git config --global user.name \"github-actions[bot]\"\n",
		"          git config --global am.keepcr true\n",
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
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))

	// Environment variables
	steps = append(steps, "        env:\n")
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ steps.setup-agent-output-env.outputs.GH_AW_AGENT_OUTPUT }}\n")

	// Add allowed domains configuration for URL sanitization in safe output handlers.
	// Without this, sanitizeContent() in safe_output_handler_manager.cjs only allows
	// default GitHub domains, causing user-configured allowed domains to be redacted.
	var domainsStr string
	if data.SafeOutputs != nil && len(data.SafeOutputs.AllowedDomains) > 0 {
		// allowed-domains: additional domains unioned with engine/network base set; supports ecosystem identifiers
		domainsStr = c.computeExpandedAllowedDomainsForSanitization(data)
	} else {
		domainsStr = c.computeAllowedDomainsForSanitization(data)
	}
	if domainsStr != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_ALLOWED_DOMAINS: %q\n", domainsStr))
	}
	// Pass GitHub server/API URLs so buildAllowedDomains() can add GHES domains dynamically
	steps = append(steps, "          GITHUB_SERVER_URL: ${{ github.server_url }}\n")
	steps = append(steps, "          GITHUB_API_URL: ${{ github.api_url }}\n")

	// Note: The project handler manager has been removed.
	// All project-related operations are now handled by the unified handler.

	// Add GH_AW_SAFE_OUTPUT_JOBS so the handler manager knows which message types are
	// handled by custom safe-output job steps and should be silently skipped rather than
	// reported as "No handler loaded for message type '...'".
	if customJobsJSON := buildCustomSafeOutputJobsJSON(data); customJobsJSON != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_JOBS: %q\n", customJobsJSON))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_SAFE_OUTPUT_JOBS env var for custom safe job types")
	}

	// Add GH_AW_SAFE_OUTPUT_SCRIPTS so the handler manager can load inline script handlers.
	// The env var maps normalized script names to their .cjs filenames in the actions folder.
	if customScriptsJSON := buildCustomSafeOutputScriptsJSON(data); customScriptsJSON != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_SCRIPTS: %q\n", customScriptsJSON))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_SAFE_OUTPUT_SCRIPTS env var for custom script handlers")
	}

	// Add GH_AW_SAFE_OUTPUT_ACTIONS so the handler manager can load custom action handlers.
	// The env var maps normalized action names to themselves (reserved for future extensibility).
	if customActionsJSON := buildCustomSafeOutputActionsJSON(data); customActionsJSON != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_ACTIONS: %q\n", customActionsJSON))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_SAFE_OUTPUT_ACTIONS env var for custom action handlers")
	}

	// Add custom safe output env vars
	c.addCustomSafeOutputEnvVars(&steps, data)

	// Add handler manager config as JSON
	c.addHandlerManagerConfigEnvVar(&steps, data)

	// Add all safe output configuration env vars (still needed by individual handlers)
	c.addAllSafeOutputConfigEnvVars(&steps, data)

	// Add extra empty commit token if create-pull-request or push-to-pull-request-branch is configured.
	// This token is used to push an empty commit after code changes to trigger CI events,
	// working around the GITHUB_TOKEN limitation where events don't trigger other workflows.
	// Only emit this env var when one of these safe outputs is actually configured.
	if usesPatchesAndCheckouts(data.SafeOutputs) {
		var ciTriggerToken string
		if data.SafeOutputs.CreatePullRequests != nil && data.SafeOutputs.CreatePullRequests.GithubTokenForExtraEmptyCommit != "" {
			ciTriggerToken = data.SafeOutputs.CreatePullRequests.GithubTokenForExtraEmptyCommit
		} else if data.SafeOutputs.PushToPullRequestBranch != nil && data.SafeOutputs.PushToPullRequestBranch.GithubTokenForExtraEmptyCommit != "" {
			ciTriggerToken = data.SafeOutputs.PushToPullRequestBranch.GithubTokenForExtraEmptyCommit
		}

		switch ciTriggerToken {
		case "app":
			steps = append(steps, "          GH_AW_CI_TRIGGER_TOKEN: ${{ steps.safe-outputs-app-token.outputs.token || '' }}\n")
			consolidatedSafeOutputsStepsLog.Print("Extra empty commit using GitHub App token")
		default:
			// Use the magic GH_AW_CI_TRIGGER_TOKEN secret (default behavior when not explicitly configured)
			steps = append(steps, fmt.Sprintf("          GH_AW_CI_TRIGGER_TOKEN: %s\n", getEffectiveCITriggerGitHubToken(ciTriggerToken)))
			consolidatedSafeOutputsStepsLog.Print("Extra empty commit using GH_AW_CI_TRIGGER_TOKEN")
		}
	}

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
	projectURL, projectToken := computeProjectURLAndToken(data.SafeOutputs)

	if projectURL != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_PROJECT_URL: %q\n", projectURL))
	}

	if projectToken != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_PROJECT_GITHUB_TOKEN: %s\n", projectToken))
	}

	// Add GH_AW_ASSIGN_TO_AGENT_TOKEN when assign-to-agent is configured OR when create-issue
	// or create-pull-request is configured with copilot in assignees. All handlers create a
	// dedicated Octokit using this token (agent token preference chain), which is required
	// because the Copilot assignment API only accepts PATs (not GitHub App tokens). This env
	// var is evaluated as a GitHub Actions expression, so it resolves to the actual token value
	// before the step runs.
	if data.SafeOutputs != nil && data.SafeOutputs.AssignToAgent != nil {
		agentTokenStr := getEffectiveCopilotCodingAgentGitHubToken(data.SafeOutputs.AssignToAgent.GitHubToken)
		//nolint:gosec // G101: False positive - this is a GitHub Actions expression template, not a hardcoded credential
		steps = append(steps, fmt.Sprintf("          GH_AW_ASSIGN_TO_AGENT_TOKEN: %s\n", agentTokenStr))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_ASSIGN_TO_AGENT_TOKEN env var for assign-to-agent handler")
	} else if data.SafeOutputs != nil && data.SafeOutputs.CreateIssues != nil && hasCopilotAssignee(data.SafeOutputs.CreateIssues.Assignees) {
		agentTokenStr := getEffectiveCopilotCodingAgentGitHubToken(data.SafeOutputs.CreateIssues.GitHubToken)
		//nolint:gosec // G101: False positive - this is a GitHub Actions expression template, not a hardcoded credential
		steps = append(steps, fmt.Sprintf("          GH_AW_ASSIGN_TO_AGENT_TOKEN: %s\n", agentTokenStr))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_ASSIGN_TO_AGENT_TOKEN env var for create-issue copilot assignment handler")
	} else if data.SafeOutputs != nil && data.SafeOutputs.CreatePullRequests != nil && hasCopilotAssignee(data.SafeOutputs.CreatePullRequests.Assignees) {
		agentTokenStr := getEffectiveCopilotCodingAgentGitHubToken(data.SafeOutputs.CreatePullRequests.GitHubToken)
		//nolint:gosec // G101: False positive - this is a GitHub Actions expression template, not a hardcoded credential
		steps = append(steps, fmt.Sprintf("          GH_AW_ASSIGN_TO_AGENT_TOKEN: %s\n", agentTokenStr))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_ASSIGN_TO_AGENT_TOKEN env var for create-pull-request copilot assignment handler")
	}

	// Add GH_AW_AGENT_SESSION_TOKEN when create-agent-session is configured.
	// The create_agent_session handler passes this token as GH_TOKEN to the gh CLI
	// (agent token preference chain), which is required because the default GITHUB_TOKEN
	// does not have permission to create agent sessions via gh agent-task create.
	if data.SafeOutputs != nil && data.SafeOutputs.CreateAgentSessions != nil {
		agentSessionTokenStr := getEffectiveCopilotCodingAgentGitHubToken(data.SafeOutputs.CreateAgentSessions.GitHubToken)
		//nolint:gosec // G101: False positive - this is a GitHub Actions expression template, not a hardcoded credential
		steps = append(steps, fmt.Sprintf("          GH_AW_AGENT_SESSION_TOKEN: %s\n", agentSessionTokenStr))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_AGENT_SESSION_TOKEN env var for create-agent-session handler")
	}

	// When create-pull-request or push-to-pull-request-branch is configured with a custom token
	// (including GitHub App), expose that token as GITHUB_TOKEN so that git CLI operations in
	// the JavaScript handlers can authenticate. The create_pull_request.cjs handler reads
	// process.env.GITHUB_TOKEN to enable dynamic repo checkout for multi-repo/cross-repo
	// scenarios (allowed-repos). Without this, the handler falls back to the default
	// repo-scoped token which lacks access to other repos.
	if usesPatchesAndCheckouts(data.SafeOutputs) {
		gitToken, isCustom := computeEffectivePRCheckoutToken(data.SafeOutputs)
		// Only override GITHUB_TOKEN when a custom token (app or PAT) is explicitly configured.
		// When no custom token is set, the default repo-scoped GITHUB_TOKEN from GitHub Actions
		// is already in the environment and overriding it with the same default is unnecessary.
		if isCustom {
			//nolint:gosec // G101: False positive - this is a GitHub Actions expression template, not a hardcoded credential
			steps = append(steps, fmt.Sprintf("          GITHUB_TOKEN: %s\n", gitToken))
			consolidatedSafeOutputsStepsLog.Printf("Adding GITHUB_TOKEN env var for cross-repo git CLI operations")
		}
	}

	// With section for github-token
	// Use the standard safe outputs token for all operations.
	// If project operations are configured, prefer the project token for the github-script client.
	// Rationale: update_project/create_project_status_update call the Projects v2 GraphQL API, which
	// cannot be accessed with the default GITHUB_TOKEN. GH_AW_PROJECT_GITHUB_TOKEN is the required
	// token for Projects v2 operations.
	steps = append(steps, "        with:\n")
	// Token precedence for the handler manager step:
	//   1. Project token (if project operations are configured) - already set above
	//   2. Safe-outputs level token (so.GitHubToken)
	//   3. Magic secret fallback via getEffectiveSafeOutputGitHubToken()
	//
	// Note: We do NOT fall back to per-output tokens (add-comment, create-issue, etc.)
	// because those are specific to their operations. The handler manager needs a
	// general-purpose token for the github-script client.
	configToken := ""
	if projectToken != "" {
		configToken = projectToken
	} else if data.SafeOutputs != nil && data.SafeOutputs.GitHubToken != "" {
		configToken = data.SafeOutputs.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, configToken)

	steps = append(steps, "          script: |\n")
	steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
	steps = append(steps, "            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	steps = append(steps, "            const { main } = require('"+SetupActionDestination+"/safe_output_handler_manager.cjs');\n")
	steps = append(steps, "            await main();\n")

	return steps
}
